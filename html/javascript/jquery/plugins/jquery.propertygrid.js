/**
 * jQuery EasyUI 1.2.6
 * 
 * Licensed under the GPL terms
 * To use it on other terms please contact us
 *
 * Copyright(c) 2009-2012 stworthy [ stworthy@gmail.com ] 
 * 
 */
(function($){
function _1(_2){
var _3=$.data(_2,"propertygrid").options;
$(_2).datagrid($.extend({},_3,{view:(_3.showGroup?_4:undefined),onClickRow:function(_5,_6){
if(_3.editIndex!=_5){
var _7=$(this).datagrid("getColumnOption","value");
_7.editor=_6.editor;
_8(_3.editIndex);
$(this).datagrid("beginEdit",_5);
$(this).datagrid("getEditors",_5)[0].target.focus();
_3.editIndex=_5;
}
_3.onClickRow.call(_2,_5,_6);
}}));
$(_2).datagrid("getPanel").panel("panel").addClass("propertygrid");
$(_2).datagrid("getPanel").find("div.datagrid-body").unbind(".propertygrid").bind("mousedown.propertygrid",function(e){
e.stopPropagation();
});
$(document).unbind(".propertygrid").bind("mousedown.propertygrid",function(){
_8(_3.editIndex);
_3.editIndex=undefined;
});
function _8(_9){
if(_9==undefined){
return;
}
var t=$(_2);
if(t.datagrid("validateRow",_9)){
t.datagrid("endEdit",_9);
}else{
t.datagrid("cancelEdit",_9);
}
};
};
$.fn.propertygrid=function(_a,_b){
if(typeof _a=="string"){
var _c=$.fn.propertygrid.methods[_a];
if(_c){
return _c(this,_b);
}else{
return this.datagrid(_a,_b);
}
}
_a=_a||{};
return this.each(function(){
var _d=$.data(this,"propertygrid");
if(_d){
$.extend(_d.options,_a);
}else{
$.data(this,"propertygrid",{options:$.extend({},$.fn.propertygrid.defaults,$.fn.propertygrid.parseOptions(this),_a)});
}
_1(this);
});
};
$.fn.propertygrid.methods={};
$.fn.propertygrid.parseOptions=function(_e){
var t=$(_e);
return $.extend({},$.fn.datagrid.parseOptions(_e),{showGroup:(t.attr("showGroup")?t.attr("showGroup")=="true":undefined)});
};
var _4=$.extend({},$.fn.datagrid.defaults.view,{render:function(_f,_10,_11){
var _12=$.data(_f,"datagrid").options;
var _13=$.data(_f,"datagrid").data.rows;
var _14=$(_f).datagrid("getColumnFields",_11);
var _15=[];
var _16=0;
var _17=this.groups;
for(var i=0;i<_17.length;i++){
var _18=_17[i];
_15.push("<div class=\"datagrid-group\" group-index="+i+" style=\"\">");
_15.push("<table cellspacing=\"0\" cellpadding=\"0\" border=\"0\" style=\"height:100%\"><tbody>");
_15.push("<tr>");
_15.push("<td style=\"border:0;\">");
if(!_11){
_15.push("<span style=\"color:#666;font-weight:bold;\">");
_15.push(_12.groupFormatter.call(_f,_18.fvalue,_18.rows));
_15.push("</span>");
}
_15.push("</td>");
_15.push("</tr>");
_15.push("</tbody></table>");
_15.push("</div>");
_15.push("<table cellspacing=\"0\" cellpadding=\"0\" border=\"0\"><tbody>");
for(var j=0;j<_18.rows.length;j++){
var cls=(_16%2&&_12.striped)?"class=\"datagrid-row-alt\"":"";
var _19=_12.rowStyler?_12.rowStyler.call(_f,_16,_18.rows[j]):"";
var _1a=_19?"style=\""+_19+"\"":"";
_15.push("<tr datagrid-row-index=\""+_16+"\" "+cls+" "+_1a+">");
_15.push(this.renderRow.call(this,_f,_14,_11,_16,_18.rows[j]));
_15.push("</tr>");
_16++;
}
_15.push("</tbody></table>");
}
$(_10).html(_15.join(""));
},onAfterRender:function(_1b){
var _1c=$.data(_1b,"datagrid").options;
var dc=$.data(_1b,"datagrid").dc;
var _1d=dc.view;
var _1e=dc.view1;
var _1f=dc.view2;
$.fn.datagrid.defaults.view.onAfterRender.call(this,_1b);
if(_1c.rownumbers||_1c.frozenColumns.length){
var _20=_1e.find("div.datagrid-group");
}else{
var _20=_1f.find("div.datagrid-group");
}
$("<td style=\"border:0\"><div class=\"datagrid-row-expander datagrid-row-collapse\" style=\"width:25px;height:16px;cursor:pointer\"></div></td>").insertBefore(_20.find("td"));
_1d.find("div.datagrid-group").each(function(){
var _21=$(this).attr("group-index");
$(this).find("div.datagrid-row-expander").bind("click",{groupIndex:_21},function(e){
if($(this).hasClass("datagrid-row-collapse")){
$(_1b).datagrid("collapseGroup",e.data.groupIndex);
}else{
$(_1b).datagrid("expandGroup",e.data.groupIndex);
}
});
});
},onBeforeRender:function(_22,_23){
var _24=$.data(_22,"datagrid").options;
var _25=[];
for(var i=0;i<_23.length;i++){
var row=_23[i];
var _26=_27(row[_24.groupField]);
if(!_26){
_26={fvalue:row[_24.groupField],rows:[row],startRow:i};
_25.push(_26);
}else{
_26.rows.push(row);
}
}
function _27(_28){
for(var i=0;i<_25.length;i++){
var _29=_25[i];
if(_29.fvalue==_28){
return _29;
}
}
return null;
};
this.groups=_25;
var _2a=[];
for(var i=0;i<_25.length;i++){
var _26=_25[i];
for(var j=0;j<_26.rows.length;j++){
_2a.push(_26.rows[j]);
}
}
$.data(_22,"datagrid").data.rows=_2a;
}});
$.extend($.fn.datagrid.methods,{expandGroup:function(jq,_2b){
return jq.each(function(){
var _2c=$.data(this,"datagrid").dc.view;
if(_2b!=undefined){
var _2d=_2c.find("div.datagrid-group[group-index=\""+_2b+"\"]");
}else{
var _2d=_2c.find("div.datagrid-group");
}
var _2e=_2d.find("div.datagrid-row-expander");
if(_2e.hasClass("datagrid-row-expand")){
_2e.removeClass("datagrid-row-expand").addClass("datagrid-row-collapse");
_2d.next("table").show();
}
$(this).datagrid("fixRowHeight");
});
},collapseGroup:function(jq,_2f){
return jq.each(function(){
var _30=$.data(this,"datagrid").dc.view;
if(_2f!=undefined){
var _31=_30.find("div.datagrid-group[group-index=\""+_2f+"\"]");
}else{
var _31=_30.find("div.datagrid-group");
}
var _32=_31.find("div.datagrid-row-expander");
if(_32.hasClass("datagrid-row-collapse")){
_32.removeClass("datagrid-row-collapse").addClass("datagrid-row-expand");
_31.next("table").hide();
}
$(this).datagrid("fixRowHeight");
});
}});
$.fn.propertygrid.defaults=$.extend({},$.fn.datagrid.defaults,{singleSelect:true,remoteSort:false,fitColumns:true,loadMsg:"",frozenColumns:[[{field:"f",width:16,resizable:false}]],columns:[[{field:"name",title:"Name",width:100,sortable:true},{field:"value",title:"Value",width:100,resizable:false}]],showGroup:false,groupField:"group",groupFormatter:function(_33){
return _33;
}});
})(jQuery);

