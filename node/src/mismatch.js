// mismatch-checker.js
import fs from "fs";
import fetch from "node-fetch"; // install with: npm install node-fetch@2

const gameIds = ["55002afe-67da-41bf-83c4-41d8eba5ad5e",
"e13be546-73c9-4482-b65b-a5aea643928e",
"7da9f3cf-238d-4cac-9191-367d65608c74",
"0c463f2b-34c8-478d-b39e-1731717d9351",
"57832e4b-c5bf-4102-b05b-3c7ad58a2bb1",
"ad93111b-3156-439a-9546-f3c34a7837d7",
"ecec1c1d-d153-4887-b5cc-bf53d0438500",
"1cf450fd-be02-4093-88a3-42f8d3cf10b9",
"b4e72736-b41a-4ca5-adcd-21cc00923a9f",
"fed1f277-2841-45f8-a150-71b639af8237",
"756d7a2b-f95b-41ec-beab-fe76316a520b",
"4ab7a681-e3fa-40f2-b2cf-0d134d67d42a",
"3e950843-f01c-48e0-a24e-a68c50b9af44",
"3cbed4fd-b65f-4362-8534-71ae893eba26",
"cd420e9b-7e30-40c3-9e19-36a71cea5db8",
"d6bc165c-3d05-403f-b637-16eec4219883",
"11e8ce9c-35c2-47a6-877f-56ab4f2d0aec"];

const publishedUrl = (id) =>
  `https://ap-game-play-service.flarie.com/v1/game/${id}`;
const stageUrl = (id) =>
  `https://ap-game-play-service.flarie.com/v1/game/${id}?readGamePkSuffix=stage`;

//https://ap-game-play-service.flarie.com/v1/game/55002afe-67da-41bf-83c4-41d8eba5ad5e
/**
 * Fetch JSON helper
 */
async function safeFetch(url) {
  try {
    const res = await fetch(url);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  } catch (err) {
    console.error("Fetch error:", url, err.message);
    return null;
  }
}

/**
 * Extract only the required fields
 */
function extractData(json) {
  if (!json || !json || !json.detail) return {};
  const d = json.detail;
  return {
    name: d.name || "",
    startDate: d.startDate || "",
    endDate: d.endDate || "",
    gameCenterId: d.gameCenterId || "",
  };
}

/**
 * Main
 */
async function run() {
  let rows = [];

  // CSV header
  rows.push([
    "gameId",
    "gameLoad(30days)",
    "StartDate(Stage)",
    "StartDate(Published)",
    "EndDate(Stage)",
    "EndDate(Published)",
    "Name(Stage)",
    "Name(Published)",
    "GameCenterId(Stage)",
    "GameCenterId(Published)",
  ]);

  for (let id of gameIds) {
    console.log("Checking game:", id);

    const [pub, stage] = await Promise.all([
      safeFetch(publishedUrl(id)),
      safeFetch(stageUrl(id)),
    ]);

    const pubData = extractData(pub);
    const stageData = extractData(stage);

    // Add CSV row (gameLoad(30days) left empty for now, unless you have API)
    rows.push([
      id,
      "", // put gameLoad value here if available
      stageData.startDate,
      pubData.startDate,
      stageData.endDate,
      pubData.endDate,
      stageData.name,
      pubData.name,
      stageData.gameCenterId,
      pubData.gameCenterId,
    ]);
  }

  const csv = rows.map((r) => r.join(",")).join("\n");
  fs.writeFileSync("mismatch_report.csv", csv);
  console.log("✅ mismatch_report.csv generated!");
}

run();
