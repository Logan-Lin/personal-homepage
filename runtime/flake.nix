{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }: let
    systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
    forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
  in {
    devShells = forAllSystems (pkgs: {
      default = pkgs.mkShell {
        packages = with pkgs; [
          uv python312
          (writeShellScriptBin "serve" ''
            python generate.py && python watch.py
          '')
          (writeShellScriptBin "build" ''
            python generate.py
          '')
        ];
        shellHook = ''
          uv sync
          source .venv/bin/activate
        '';
      };
    });
  };
}
