{ pkgs ? import <nixpkgs> {}}:

pkgs.mkShell {
  packages = with pkgs; [
    python312
    uv
  ];

  shellHook = let
    venvPath = "$HOME/.venv/homepage";
  in ''
    # Set uv to use specific virtual environment path
    export UV_PROJECT_ENVIRONMENT=${venvPath}
    
    # Install dependencies with uv
    uv sync

    # Activate the virtual environment
    source ${venvPath}/bin/activate

    # Define aliases
    alias serve="python generate.py && python watch.py"
    
    echo "Available commands:"
    echo "  serve      - Watch and rebuild on changes"
  '';
}
