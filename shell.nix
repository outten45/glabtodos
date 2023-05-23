# using with direnv, add this to .envrc:
# 
#    use nix
{ pkgs ? import <nixpkgs> { } }:

let

in pkgs.mkShell {
  # nativeBuildInputs is usually what you want -- tools you need to run
  nativeBuildInputs = [
    pkgs.buildPackages.go
    pkgs.buildPackages.gnumake
    pkgs.buildPackages.gcc
    pkgs.buildPackages.sqlite-interactive
    pkgs.buildPackages.readline
    pkgs.buildPackages.gopls
  ];
  shellHook = ''
    echo "Starting nix-shell with fish..."
  '';

}
