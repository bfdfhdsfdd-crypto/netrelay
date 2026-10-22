const fs = require("fs")
const manifest = require("./wails.json")
const version = process.argv[2].replace("v", "").split("-")[0]

if (!version) {
    console.log("No version supplied")
    process.exit(1)
}

// update version in wails.json
manifest.info.productVersion = version
const data = JSON.stringify(manifest, null, 2)
fs.writeFileSync("wails.json", data)

// Set the window title to the version being packaged.
const mainGoPath = "./main.go"
const mainGoContent = fs.readFileSync(mainGoPath, "utf8")
const titlePattern = /Title:\s+"NetRelay(?: v\d+\.\d+\.\d+)?"/
if (!titlePattern.test(mainGoContent)) {
    console.error("Could not find the NetRelay window title in main.go")
    process.exit(1)
}
const newMainGoContent = mainGoContent.replace(titlePattern, `Title: "NetRelay v${version}"`)
fs.writeFileSync(mainGoPath, newMainGoContent)

console.log(`Updated build version to ${version}`)