{ pkgs ? import <nixpkgs> {}}:

pkgs.mkShell {
  packages = with pkgs; [
    python312
    uv
  ];

  shellHook = let
    venvPath = "$HOME/.venv/homepage";
    remoteHost = "vps";
  in ''
    # Set uv to use specific virtual environment path
    export UV_PROJECT_ENVIRONMENT=${venvPath}
    
    # Install dependencies with uv
    uv sync

    # Activate the virtual environment
    source ${venvPath}/bin/activate

    # Define aliases
    alias serve="python generate.py && python watch.py"
    alias build="python generate.py"
    alias sync="python generate.py && rsync -avP --delete ./dist/* ${remoteHost}:~/www/homepage"
    
    echo "Available commands:"
    echo "  serve      - Watch and rebuild on changes (dev mode)"
    echo "  build      - Generate the static site"
    echo "  sync       - Build and sync with remote production server"

  '';
}
