/* 
 * Objects/constants stored in /js/consts.js
 */

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
}