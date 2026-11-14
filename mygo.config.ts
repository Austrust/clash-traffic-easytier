import { defineConfig } from "mygo-cli";

// Local feature builds must not be replaced by an upstream automatic update.
const easyTierBuild = process.env.CLASH_TRAFFIC_EASYTIER_BUILD === "1";

export default defineConfig({
  name: "Clash Traffic",
  identifier: "dev.lufficc.clashtraffic",
  version: easyTierBuild ? "0.1.4-easytier.2" : "0.1.4",
  // `mygo dev` runs devCommand and loads devUrl; `mygo build` runs
  // buildCommand and embeds frontendDist into the app.
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
  // Apps update themselves from the GitHub releases, signed with the key
  // whose public half is here. Release builds need the secret half in
  // MYGO_UPDATER_PRIVATE_KEY; see CHANGELOG.md for the release notes.
  updates: easyTierBuild ? undefined : {
    publicKey: "XvFzwWpm0jjixED5INf3tnNIaCOWt7qKW3ux7cPmiN0=",
    github: "lufficc/clash-traffic",
  },
  macos: {
    // Go 1.27 links for macOS 13 and later, so the app cannot run on 12.
    minimumSystemVersion: "13.0",
  },
});
