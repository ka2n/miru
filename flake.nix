{
  description = "miru - A CLI tool for viewing package documentation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.0.21";
      in
      {
        packages = {
          miru = pkgs.buildGoModule {
            pname = "miru";
            inherit version;
            src = ./.;
            vendorHash = "sha256-Q2n0TwPMYwC6lbQfSw9xgM2eMs0HLcKE9uXh7Tt7RG0=";
            subPackages = [ "cmd/miru" ];

            ldflags = [
              "-s"
              "-w"
            ];

            meta = {
              description = "A CLI tool for viewing package documentation";
              homepage = "https://github.com/ka2n/miru";
              mainProgram = "miru";
            };
          };

          default = self.packages.${system}.miru;
        };
      }
    );
}
