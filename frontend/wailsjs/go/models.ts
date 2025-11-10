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

}

