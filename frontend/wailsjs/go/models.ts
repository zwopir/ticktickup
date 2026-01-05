export namespace main {
	
	export class ImportResult {
	    success: boolean;
	    message: string;
	    imported: number;
	    failed: number;
	    errors?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.imported = source["imported"];
	        this.failed = source["failed"];
	        this.errors = source["errors"];
	    }
	}
	export class ProjectInfo {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

