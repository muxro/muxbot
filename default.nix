let 
    pkgs = import <nixpkgs> {};
in
pkgs.buildGoPackage {
    name = "muxbot";
    goPackagePath = "gitlab.com/muxro/muxbot";
    src = ./.;
}