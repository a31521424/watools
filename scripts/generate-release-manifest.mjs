import {createHash} from "node:crypto"
import {promises as fs} from "node:fs"
import path from "node:path"

const artifactsDir = process.env.ARTIFACTS_DIR
const releaseVersion = process.env.RELEASE_VERSION
const releaseTag = process.env.RELEASE_TAG
const repoSlug = process.env.REPO_SLUG || "a31521424/watools"
const releaseMetadataFile = process.env.RELEASE_METADATA_FILE

if (!artifactsDir || !releaseVersion || !releaseTag || !releaseMetadataFile) {
    throw new Error("ARTIFACTS_DIR, RELEASE_VERSION, RELEASE_TAG and RELEASE_METADATA_FILE are required")
}

const releaseMetadata = JSON.parse(await fs.readFile(releaseMetadataFile, "utf8"))

const expectedAssets = [
    {
        platformKey: "windows-amd64",
        os: "windows",
        arch: "amd64",
        installerType: "nsis",
        fileName: `watools_${releaseVersion}_windows_amd64_installer.exe`,
    },
    {
        platformKey: "darwin-universal",
        os: "darwin",
        arch: "universal",
        installerType: "app-zip",
        fileName: `watools_${releaseVersion}_macos_universal.zip`,
    },
]

const files = await collectFiles(artifactsDir)
const fileIndex = new Map(files.map(filePath => [path.basename(filePath), filePath]))

const manifest = {
    version: releaseVersion,
    releaseTag,
    releaseName: releaseMetadata.name || releaseMetadata.tagName || releaseTag,
    releaseUrl: releaseMetadata.url || `https://github.com/${repoSlug}/releases/tag/${releaseTag}`,
    publishedAt: releaseMetadata.publishedAt || new Date().toISOString(),
    notes: releaseMetadata.body || "",
    generatedAt: new Date().toISOString(),
    platforms: {},
}

const checksumLines = []
for (const asset of expectedAssets) {
    const absolutePath = fileIndex.get(asset.fileName)
    if (!absolutePath) {
        throw new Error(`Missing expected release asset: ${asset.fileName}`)
    }

    const stat = await fs.stat(absolutePath)
    const sha256 = await checksumFile(absolutePath)
    checksumLines.push(`${sha256}  ${asset.fileName}`)

    manifest.platforms[asset.platformKey] = {
        os: asset.os,
        arch: asset.arch,
        assetName: asset.fileName,
        url: `https://github.com/${repoSlug}/releases/download/${releaseTag}/${asset.fileName}`,
        sha256,
        size: stat.size,
        installerType: asset.installerType,
    }
}

await fs.writeFile(
    path.join(artifactsDir, "watools_latest.json"),
    `${JSON.stringify(manifest, null, 2)}\n`,
    "utf8",
)

await fs.writeFile(
    path.join(artifactsDir, "SHA256SUMS.txt"),
    `${checksumLines.sort().join("\n")}\n`,
    "utf8",
)

async function collectFiles(rootDir) {
    const entries = await fs.readdir(rootDir, {withFileTypes: true})
    const results = []

    for (const entry of entries) {
        const absolutePath = path.join(rootDir, entry.name)
        if (entry.isDirectory()) {
            results.push(...await collectFiles(absolutePath))
            continue
        }
        if (entry.isFile()) {
            results.push(absolutePath)
        }
    }

    return results
}

async function checksumFile(filePath) {
    const content = await fs.readFile(filePath)
    return createHash("sha256").update(content).digest("hex")
}
