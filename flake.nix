{
  description = "Pulumi provider for Chatto";

  inputs = {
    # Source of the generated API client and permission catalog; keep in step with the supported Chatto version
    chatto = {
      url = "github:chattocorp/chatto/v0.6.0";
      flake = false;
    };
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      chatto,
      nixpkgs,
      self,
      treefmt-nix,
    }:
    let
      forAllSystems =
        f:
        nixpkgs.lib.genAttrs [
          "aarch64-darwin"
          "aarch64-linux"
          "x86_64-linux"
        ] (system: f nixpkgs.legacyPackages.${system});

      treefmt = forAllSystems (
        pkgs:
        treefmt-nix.lib.evalModule pkgs {
          projectRootFile = "flake.nix";

          programs = {
            actionlint.enable = true;
            deadnix.enable = true;
            flake-edit.enable = true;
            gofmt.enable = true;
            nixf-diagnose.enable = true;
            nixfmt.enable = true;
            oxfmt.enable = true;
            statix.enable = true;
            toml-sort.enable = true;
          };

          settings.global.excludes = [
            "LICENSES/*"
            "internal/pb/**"
            "provider/permissions_gen.go"
          ];
        }
      );

      # Development commands, on PATH in `nix develop` and runnable with `nix run .#<name>`
      scripts = forAllSystems (
        pkgs:
        let
          script =
            name: runtimeInputs: text:
            pkgs.writeShellApplication {
              inherit name runtimeInputs;
              text = ''
                cd "$(git rev-parse --show-toplevel)"
                export GOFLAGS="''${GOFLAGS:-} -trimpath"
                ${text}
              '';
              runtimeEnv.CHATTO_SRC = chatto;
            };

          go = [
            pkgs.git
            pkgs.go
          ];
        in
        rec {
          build = script "build" go ''
            go build -o bin/pulumi-resource-chatto ./cmd/pulumi-resource-chatto
          '';

          # Regenerates code from the Chatto source pinned in flake.lock
          generate =
            script "generate"
              (
                go
                ++ [
                  pkgs.buf
                  pkgs.bun
                  pkgs.protoc-gen-connect-go
                  pkgs.protoc-gen-go
                ]
              )
              ''
                rm -rf internal/pb

                buf generate "$CHATTO_SRC/proto" --template buf.gen.yaml \
                  --path "$CHATTO_SRC/proto/chatto/admin/v1" --path "$CHATTO_SRC/proto/chatto/api/v1"

                bun tools/gen-permissions.ts > provider/permissions_gen.go
                gofmt -w provider/permissions_gen.go
              '';

          lint =
            script "lint"
              (
                go
                ++ [
                  pkgs.go-tools
                  pkgs.gotools
                  treefmt.${pkgs.stdenv.hostPlatform.system}.config.build.wrapper
                ]
              )
              ''
                treefmt --ci || { echo 'Fix with: treefmt' >&2; exit 1; }
                go vet ./...
                staticcheck -checks U1000 ./...

                # modernize exits with 3 when it reports diagnostics; protoc-gen-go owns the *.pb.go files
                status=0
                output=$(modernize ./... 2>&1) || status=$?
                if [ "$status" -ne 0 ] && [ "$status" -ne 3 ]; then
                  echo "$output" >&2
                  exit "$status"
                fi

                findings=$(echo "$output" | grep -v '\.pb\.go:' | grep -v '^$' || true)
                if [ -n "$findings" ]; then
                  echo "$findings" >&2
                  echo 'Fix with: modernize -fix ./...' >&2
                  exit 1
                fi
              '';

          run-tests = script "run-tests" go ''
            go test -shuffle="''${PULUMI_CHATTO_GO_TEST_SHUFFLE:-on}" -timeout 5m ./...
          '';

          schema = script "schema" [ pkgs.pulumi-bin ] ''
            ${pkgs.lib.getExe build}
            pulumi package get-schema ./bin/pulumi-resource-chatto > schema.json
          '';

          sdk = script "sdk" [ pkgs.pulumi-bin ] ''
            ${pkgs.lib.getExe build}
            for language in go nodejs python dotnet java; do
              pulumi package gen-sdk ./bin/pulumi-resource-chatto --language "$language" --out sdk
            done
          '';

          # Everything CI runs
          check =
            script "check"
              [
                pkgs.actionlint
                pkgs.git
                pkgs.reuse
              ]
              ''
                ${pkgs.lib.getExe generate}

                git diff --exit-code -- internal/pb provider/permissions_gen.go ||
                  { echo 'Generated code is stale. Run generate.' >&2; exit 1; }

                reuse lint
                actionlint
                ${pkgs.lib.getExe lint}
                ${pkgs.lib.getExe run-tests}
              '';
        }
      );
    in
    {
      apps = forAllSystems (
        pkgs:
        builtins.mapAttrs (_: drv: {
          type = "app";
          program = pkgs.lib.getExe drv;
        }) scripts.${pkgs.stdenv.hostPlatform.system}
      );

      checks = forAllSystems (pkgs: {
        formatting = treefmt.${pkgs.stdenv.hostPlatform.system}.config.build.check self;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages =
            with pkgs;
            [
              actionlint
              buf
              bun
              go
              go-tools
              goreleaser
              gotools
              protoc-gen-connect-go
              protoc-gen-go
              pulumi-bin
              reuse
              shellcheck
              treefmt.${stdenv.hostPlatform.system}.config.build.wrapper
            ]
            ++ builtins.attrValues scripts.${pkgs.stdenv.hostPlatform.system};

          CHATTO_SRC = chatto;
          GOFLAGS = "-trimpath";
        };
      });

      formatter = forAllSystems (pkgs: treefmt.${pkgs.stdenv.hostPlatform.system}.config.build.wrapper);

      packages = forAllSystems (pkgs: {
        default = pkgs.buildGoModule {
          pname = "pulumi-resource-chatto";
          version = self.shortRev or "dirty";
          src = self;
          subPackages = [ "cmd/pulumi-resource-chatto" ];
          vendorHash = "sha256-gx4Zh2jdxYXQLGZ2JUKDSUhau0u7cI+pu6+Epkxw09Y=";
          env.CGO_ENABLED = "0";
          # main.version stays at its semver default: Pulumi rejects a commit hash as a plugin version
          ldflags = [
            "-s"
            "-w"
          ];
          meta.mainProgram = "pulumi-resource-chatto";
        };
      });
    };
}
