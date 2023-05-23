# using nix flake with an existing shell.nix
# https://nixos.wiki/wiki/Flakes#Super_fast_nix-shell
# 
# using with direnv, add the following to .envrc:
#   
#   use flake .
# 
{
  description = "go and packages for nix";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system};
      in { devShells.default = import ./shell.nix { inherit pkgs; }; });
}

