export namespace models {
	
	export class Plugin {
	    id: string;
	    packageID: string;
	    name: string;
	    version: string;
	    description: string;
	    author: string;
	    internal: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Plugin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.packageID = source["packageID"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.author = source["author"];
	        this.internal = source["internal"];
	    }
	}

}

