{ pkgs ? import <nixpkgs> {}, dev ? false, restartRemote ? false }:

pkgs.mkShell {
  packages = with pkgs; [
    python312
    uv
  ];

  shellHook = let
    remoteHost = "personal-vps";
  in ''
    # Install dependencies with uv
    uv sync ${if dev then "--group dev" else ""}
    
    # Activate the virtual environment
    source .venv/bin/activate

    python generate.py

    ${if dev then ''
      python watch.py && exit
    '' else ''
      rsync -avP --delete ./{dist,compose.yml} ${remoteHost}:/root/homepage/

      ${if restartRemote then ''
        ssh ${remoteHost} "cd /root/homepage && docker compose down && docker compose up -d --remove-orphans"
      '' else ""}
      exit
    ''}
  '';
}
