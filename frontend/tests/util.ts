export interface ResponsePolyfill<T> {
	ok: boolean;
	headers: undefined;
	redirected: false;
	status: number;
	statusText: string;
	type: "basic" | "cors" | "default" | "error" | "opaque" | "opaqueredirect";
	url: string;
	clone: () => ResponsePolyfill<T>;
	body?: ReadableStream<Uint8Array<ArrayBuffer>>,
	bodyUsed: boolean,
	arrayBuffer: () => Promise<ArrayBuffer>;
	blob: () => Promise<Blob>;
	bytes: () => Promise<Uint8Array<ArrayBuffer>>;
	formData: () => Promise<FormData>;
	json: () => Promise<T>;
	text: () => Promise<string>;
}

export interface FormDataObject {
	[key: string]: string|number;
}

export type MockContentType  = "application/json"|"text/plain"|"text/html";

export class MockResponse<T> implements ResponsePolyfill<T> {
	ok: boolean;
	headers: undefined;
	redirected: false;
	status: number;
	statusText: string;
	type: "basic" | "cors" | "default" | "error" | "opaque" | "opaqueredirect";
	url: string;
	bodyUsed: boolean = false;
	
	contentType: MockContentType;
	private _formData: FormData;
	private _text: string;
	
	constructor(url: string, body:string, contentType: MockContentType, ok = true, status = 200, statusText = "ok") {
		this.url = "";
		this._formData = new FormData();
		this.ok = ok;
		this.status = status;
		this.statusText = statusText;
		this.type = "default";
		this._text = body;
		this.contentType = contentType;
		this.headers = undefined;
	}
	body?: ReadableStream<Uint8Array<ArrayBuffer>>;
	arrayBuffer(): Promise<ArrayBuffer> {
		let encoder = new TextEncoder();
		return Promise.resolve(encoder.encode(this._text).buffer);
	}
	blob(): Promise<Blob> {
		return Promise.resolve(new Blob([this._text], {type: this.contentType}));
	}
	bytes(): Promise<Uint8Array<ArrayBuffer>> {
		let encoder = new TextEncoder();
		return Promise.resolve(encoder.encode(this._text));
	}
	formData(): Promise<FormData> {
		return Promise.resolve(this._formData);
	}
	json(): Promise<T> {
		return Promise.resolve(JSON.parse(this._text) as T);
	}
	text(): Promise<string> {
		return Promise.resolve(this._text);
	}
	clone(): ResponsePolyfill<T> {
		return this;
	}
}
