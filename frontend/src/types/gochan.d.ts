/* 
 * Objects/constants stored in /js/consts.js
 */

declare interface StaffAction {
	id: string;
	title: string;
	perms: number;
	jsonOutput: number;
}

declare interface StaffInfo {
	ID: number;
	Username: string;
	Rank: number;
	AddedOn: string;
	LastActive: string;
}

declare interface Style {
	Name: string;
	Filename: string;
}


declare let styles: Style[];
declare let defaultStyle: string;
declare let webroot: string;
declare let serverTZ: number;

interface Window {
	styles: Style[];
	defaultStyle: string;
	webroot: string;
	serverTZ: number;

	openQR: () => void;
	closeQR: () => void;
	toTop: () => void;
	toBottom: () => void;
}