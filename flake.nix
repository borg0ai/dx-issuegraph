{
  description = "issuegraph (issuegraph) - An issue tracker designed for AI-supervised coding workflows";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
  };

  outputs =
    {
      self,
      nixpkgs,
      ...
    }:
    let
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];

      overlay = import ./overlay.nix self;

      forAllSystems =
        f:
        nixpkgs.lib.genAttrs systems (
          system:
          f (
            import nixpkgs {
              inherit system;
              overlays = [ overlay ];
            }
          )
        );
    in
    {
      overlays.default = overlay;

      packages = forAllSystems (import ./packages.nix);

      apps = nixpkgs.lib.genAttrs systems (
        system:
        rec {
          issuegraph = {
            type = "app";
            program = "${self.packages.${system}.default}/bin/issuegraph";
          };
          default = issuegraph;
        }
      );

      devShells = forAllSystems (
        pkgs:
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go_1_26
              git
              gopls
              gotools
              golangci-lint
            ];
            shellHook = ''
              echo "issuegraph development shell"
              echo "Go version: $(go version)"
            '';
          };
        }
      );
    };
}
