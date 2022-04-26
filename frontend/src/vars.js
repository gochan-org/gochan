import jquery from "jquery";
export default (window.$ = window.jQuery = jquery);

export const downArrow = "&#9660;";
export const upArrow = "&#9650;";
export const opRegex = /(\d+)(p(\d)+)?.html$/;

export let dropdownDivCreated = false;
