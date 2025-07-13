export namespace models {
	
	export class Command {
	    name: string;
	    description: string;
	    category: string;
	    path: string;
	    iconPath: string;
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new Command(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.path = source["path"];
	        this.iconPath = source["iconPath"];
	        this.id = source["id"];
	    }
	}

}

