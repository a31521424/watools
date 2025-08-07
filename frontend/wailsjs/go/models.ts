export namespace watcher {
	
	export class WatcherMetrics {
	    eventsProcessed: number;
	    eventsDropped: number;
	    errorsCount: number;
	    eventsByType: Record<string, number>;
	    // Go type: time
	    lastEventTime: any;
	    // Go type: time
	    watcherStartTime: any;
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
	        this.lastEventTime = this.convertValues(source["lastEventTime"], null);
	        this.watcherStartTime = this.convertValues(source["watcherStartTime"], null);
	        this.processingDuration = source["processingDuration"];
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

