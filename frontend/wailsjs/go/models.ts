export namespace app {
	
	export class HotkeyConfigAPI {
	    id: string;
	    name: string;
	    hotkey: string;
	
	    static createFrom(source: any = {}) {
	        return new HotkeyConfigAPI(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.hotkey = source["hotkey"];
	    }
	}

}

export namespace watcher {
	
	export class WatcherMetrics {
	    eventsProcessed: number;
	    eventsDropped: number;
	    errorsCount: number;
	    eventsByType: Record<string, number>;
	    lastEventTime: string;
	    watcherStartTime: string;
	    processingDuration: number;
	
	    static createFrom(source: any = {}) {
	        return new WatcherMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.eventsProcessed = source["eventsProcessed"];
	        this.eventsDropped = source["eventsDropped"];
	        this.errorsCount = source["errorsCount"];
	        this.eventsByType = source["eventsByType"];
	        this.lastEventTime = source["lastEventTime"];
	        this.watcherStartTime = source["watcherStartTime"];
	        this.processingDuration = source["processingDuration"];
	    }
	}

}

