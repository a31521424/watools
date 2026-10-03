export namespace app {
	
	export class ClipboardContent {
	    types: string[];
	    contentType: string;
	    hasText: boolean;
	    hasImage: boolean;
	    hasFiles: boolean;
	    text?: string;
	    imageBase64?: string;
	    files?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ClipboardContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.types = source["types"];
	        this.contentType = source["contentType"];
	        this.hasText = source["hasText"];
	        this.hasImage = source["hasImage"];
	        this.hasFiles = source["hasFiles"];
	        this.text = source["text"];
	        this.imageBase64 = source["imageBase64"];
	        this.files = source["files"];
	    }
	}
	export class HotkeyEnvironmentStatus {
	    secureEventInputEnabled: boolean;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new HotkeyEnvironmentStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secureEventInputEnabled = source["secureEventInputEnabled"];
	        this.note = source["note"];
	    }
	}

}

export namespace logger {
	
	export class LogFileInfo {
	    name: string;
	    path: string;
	    size: number;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new LogFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class LogRecord {
	    id: string;
	    file: string;
	    filePath: string;
	    line: number;
	    raw: string;
	    parseStatus: string;
	    timestamp?: string;
	    level?: string;
	    message: string;
	    source?: string;
	    pluginId?: string;
	    error?: string;
	    fields?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new LogRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file = source["file"];
	        this.filePath = source["filePath"];
	        this.line = source["line"];
	        this.raw = source["raw"];
	        this.parseStatus = source["parseStatus"];
	        this.timestamp = source["timestamp"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.source = source["source"];
	        this.pluginId = source["pluginId"];
	        this.error = source["error"];
	        this.fields = source["fields"];
	    }
	}
	export class LogQueryResult {
	    records: LogRecord[];
	    nextCursor?: string;
	    availableSources: string[];
	    availablePluginIds: string[];
	    totalMatched: number;
	
	    static createFrom(source: any = {}) {
	        return new LogQueryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.records = this.convertValues(source["records"], LogRecord);
	        this.nextCursor = source["nextCursor"];
	        this.availableSources = source["availableSources"];
	        this.availablePluginIds = source["availablePluginIds"];
	        this.totalMatched = source["totalMatched"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace update {
	
	export class InstallResult {
	    status: string;
	    message: string;
	    openPath: string;
	    shouldQuit: boolean;
	    requiresManualInstall: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.openPath = source["openPath"];
	        this.shouldQuit = source["shouldQuit"];
	        this.requiresManualInstall = source["requiresManualInstall"];
	    }
	}
	export class UpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    releaseTag: string;
	    releaseName: string;
	    releaseUrl: string;
	    publishedAt: string;
	    notes: string;
	    platformKey: string;
	    platformLabel: string;
	    assetName: string;
	    assetUrl: string;
	    assetSha256: string;
	    assetSize: number;
	    installerType: string;
	    hasUpdate: boolean;
	    downloadedPath: string;
	    requiresManualInstall: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.releaseTag = source["releaseTag"];
	        this.releaseName = source["releaseName"];
	        this.releaseUrl = source["releaseUrl"];
	        this.publishedAt = source["publishedAt"];
	        this.notes = source["notes"];
	        this.platformKey = source["platformKey"];
	        this.platformLabel = source["platformLabel"];
	        this.assetName = source["assetName"];
	        this.assetUrl = source["assetUrl"];
	        this.assetSha256 = source["assetSha256"];
	        this.assetSize = source["assetSize"];
	        this.installerType = source["installerType"];
	        this.hasUpdate = source["hasUpdate"];
	        this.downloadedPath = source["downloadedPath"];
	        this.requiresManualInstall = source["requiresManualInstall"];
	        this.message = source["message"];
	    }
	}

}

