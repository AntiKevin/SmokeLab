export namespace logs {
	
	export class LevelCount {
	    level: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new LevelCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.count = source["count"];
	    }
	}
	export class LogSource {
	    kind: string;
	    name: string;
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new LogSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.id = source["id"];
	    }
	}
	export class LogFilter {
	    search?: string;
	    levels: string[];
	    applications: string[];
	    sources: LogSource[];
	    // Go type: time
	    from?: any;
	    // Go type: time
	    to?: any;
	
	    static createFrom(source: any = {}) {
	        return new LogFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.search = source["search"];
	        this.levels = source["levels"];
	        this.applications = source["applications"];
	        this.sources = this.convertValues(source["sources"], LogSource);
	        this.from = this.convertValues(source["from"], null);
	        this.to = this.convertValues(source["to"], null);
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
	export class ListLogsRequest {
	    filter: LogFilter;
	    page: number;
	    pageSize: number;
	    sortBy: string;
	    sortDirection: string;
	
	    static createFrom(source: any = {}) {
	        return new ListLogsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = this.convertValues(source["filter"], LogFilter);
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.sortBy = source["sortBy"];
	        this.sortDirection = source["sortDirection"];
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
	
	export class LogOverview {
	    total: number;
	    byLevel: LevelCount[];
	    applications: string[];
	    sources: LogSource[];
	    // Go type: time
	    oldestTimestamp?: any;
	    // Go type: time
	    newestTimestamp?: any;
	
	    static createFrom(source: any = {}) {
	        return new LogOverview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.byLevel = this.convertValues(source["byLevel"], LevelCount);
	        this.applications = source["applications"];
	        this.sources = this.convertValues(source["sources"], LogSource);
	        this.oldestTimestamp = this.convertValues(source["oldestTimestamp"], null);
	        this.newestTimestamp = this.convertValues(source["newestTimestamp"], null);
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
	export class LogRecord {
	    id: number;
	    // Go type: time
	    timestamp: any;
	    level: string;
	    message: string;
	    application: string;
	    source: LogSource;
	    lineNumber: number;
	    // Go type: time
	    capturedAt: any;
	    params: string;
	
	    static createFrom(source: any = {}) {
	        return new LogRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.level = source["level"];
	        this.message = source["message"];
	        this.application = source["application"];
	        this.source = this.convertValues(source["source"], LogSource);
	        this.lineNumber = source["lineNumber"];
	        this.capturedAt = this.convertValues(source["capturedAt"], null);
	        this.params = source["params"];
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
	export class LogPage {
	    items: LogRecord[];
	    total: number;
	    page: number;
	    pageSize: number;
	    totalPages: number;
	
	    static createFrom(source: any = {}) {
	        return new LogPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], LogRecord);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.totalPages = source["totalPages"];
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

