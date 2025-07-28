{
  inputs,
  writeShellApplication,
  ...
}:

writeShellApplication {
  name = "getDB";
  text = ''
    cat "${inputs.wowsims}/assets/database/db.json"
  '';
}