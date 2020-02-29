let 
    pkgs = import <nixpkgs> {};
    nix-lang-eval = pkgs.fetchFromGitHub {
            owner = "noonien";
            repo = "nix-lang-eval";
            rev = "8283df284ff351810c22d59c78bbfdd425c3e218";
            sha256 = "1mipl4sdypcfdamhyg3scigsgchwg0aj5p6p4zvq24hhs5bxxxvh";
        };
in
pkgs.buildGoPackage {
    name = "muxbot";
    goPackagePath = "gitlab.com/muxro/muxbot";
    src = ./.;
}