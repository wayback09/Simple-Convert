export namespace main {
	
	export class ConversionParams {
	    inputPath: string;
	    outputPath: string;
	    targetFormat: string;
	    resolutionScale: string;
	    qualityPreset: string;
	    enableTrim: boolean;
	    trimStart: string;
	    trimEnd: string;
	    stripAudio: boolean;
	    extractAudio: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConversionParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPath = source["inputPath"];
	        this.outputPath = source["outputPath"];
	        this.targetFormat = source["targetFormat"];
	        this.resolutionScale = source["resolutionScale"];
	        this.qualityPreset = source["qualityPreset"];
	        this.enableTrim = source["enableTrim"];
	        this.trimStart = source["trimStart"];
	        this.trimEnd = source["trimEnd"];
	        this.stripAudio = source["stripAudio"];
	        this.extractAudio = source["extractAudio"];
	    }
	}
	export class MediaInfo {
	    path: string;
	    size: number;
	    format: string;
	    duration: number;
	    videoCodec: string;
	    resolution: string;
	    frameRate: string;
	    audioCodec: string;
	    audioChannels: number;
	    hasVideo: boolean;
	    hasAudio: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	        this.format = source["format"];
	        this.duration = source["duration"];
	        this.videoCodec = source["videoCodec"];
	        this.resolution = source["resolution"];
	        this.frameRate = source["frameRate"];
	        this.audioCodec = source["audioCodec"];
	        this.audioChannels = source["audioChannels"];
	        this.hasVideo = source["hasVideo"];
	        this.hasAudio = source["hasAudio"];
	    }
	}

}

