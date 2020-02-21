let 
    pkgs = import <nixpkgs> {};
in
pkgs.stdenv.mkDerivation {
    name = "muxbot";
    src = ./.;
    buildInputs = with pkgs; [ go ];
    buildPhase = ''
        pwd
        go build -v -mod=vendor
    '';
    installPhase = ''
        mkdir -p $out/bin
        cp muxbot $out/bin
    '';
}