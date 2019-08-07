let
  pkgs = import <nixpkgs> {};
  stdenv = pkgs.stdenv;


in stdenv.mkDerivation {
  name = "env";
  buildInputs = [
                  pkgs.doctl
                  pkgs.go
                  pkgs.kubectl
                  pkgs.kubectx
                ];
}
