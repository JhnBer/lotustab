const fs = require("fs");
const path = require("path");

const version = process.env.APP_VERSION;
if (!version) {
  console.error("APP_VERSION is required");
  process.exit(1);
}

const dir = __dirname;
const data = JSON.parse(fs.readFileSync(path.join(dir, "info.json"), "utf8"));
data.fixed.file_version = version;
data.info["0000"].ProductVersion = version;
fs.writeFileSync(path.join(dir, "info.generated.json"), JSON.stringify(data, null, "\t") + "\n");
