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
var _3=$.data(_2,"pagination").options;
var _4=$(_2).addClass("pagination").empty();
var t=$("<table cellspacing=\"0\" cellpadding=\"0\" border=\"0\"><tr></tr></table>").appendTo(_4);
var tr=$("tr",t);
if(_3.showPageList){
var ps=$("<select class=\"pagination-page-list\"></select>");
for(var i=0;i<_3.pageList.length;i++){
var _5=$("<option></option>").text(_3.pageList[i]).appendTo(ps);
if(_3.pageList[i]==_3.pageSize){
_5.attr("selected","selected");
}
}
$("<td></td>").append(ps).appendTo(tr);
_3.pageSize=parseInt(ps.val());
$("<td><div class=\"pagination-btn-separator\"></div></td>").appendTo(tr);
}
$("<td><a href=\"javascript:void(0)\" icon=\"pagination-first\"></a></td>").appendTo(tr);
$("<td><a href=\"javascript:void(0)\" icon=\"pagination-prev\"></a></td>").appendTo(tr);
$("<td><div class=\"pagination-btn-separator\"></div></td>").appendTo(tr);
$("<span style=\"padding-left:6px;\"></span>").html(_3.beforePageText).wrap("<td></td>").parent().appendTo(tr);
$("<td><input class=\"pagination-num\" type=\"text\" value=\"1\" size=\"2\"></td>").appendTo(tr);
$("<span style=\"padding-right:6px;\"></span>").wrap("<td></td>").parent().appendTo(tr);
$("<td><div class=\"pagination-btn-separator\"></div></td>").appendTo(tr);
$("<td><a href=\"javascript:void(0)\" icon=\"pagination-next\"></a></td>").appendTo(tr);
$("<td><a href=\"javascript:void(0)\" icon=\"pagination-last\"></a></td>").appendTo(tr);
if(_3.showRefresh){
$("<td><div class=\"pagination-btn-separator\"></div></td>").appendTo(tr);
$("<td><a href=\"javascript:void(0)\" icon=\"pagination-load\"></a></td>").appendTo(tr);
}
if(_3.buttons){
$("<td><div class=\"pagination-btn-separator\"></div></td>").appendTo(tr);
for(var i=0;i<_3.buttons.length;i++){
var _6=_3.buttons[i];
if(_6=="-"){
$("<td><div class=\"pagination-btn-separator\"></div></td>").appendTo(tr);
}else{
var td=$("<td></td>").appendTo(tr);
$("<a href=\"javascript:void(0)\"></a>").addClass("l-btn").css("float","left").text(_6.text||"").attr("icon",_6.iconCls||"").bind("click",eval(_6.handler||function(){
})).appendTo(td).linkbutton({plain:true});
}
}
}
$("<div class=\"pagination-info\"></div>").appendTo(_4);
$("<div style=\"clear:both;\"></div>").appendTo(_4);
$("a[icon^=pagination]",_4).linkbutton({plain:true});
_4.find("a[icon=pagination-first]").unbind(".pagination").bind("click.pagination",function(){
if(_3.pageNumber>1){
_b(_2,1);
}
});
_4.find("a[icon=pagination-prev]").unbind(".pagination").bind("click.pagination",function(){
if(_3.pageNumber>1){
_b(_2,_3.pageNumber-1);
}
});
_4.find("a[icon=pagination-next]").unbind(".pagination").bind("click.pagination",function(){
var _7=Math.ceil(_3.total/_3.pageSize);
if(_3.pageNumber<_7){
_b(_2,_3.pageNumber+1);
}
});
_4.find("a[icon=pagination-last]").unbind(".pagination").bind("click.pagination",function(){
var _8=Math.ceil(_3.total/_3.pageSize);
if(_3.pageNumber<_8){
_b(_2,_8);
}
});
_4.find("a[icon=pagination-load]").unbind(".pagination").bind("click.pagination",function(){
if(_3.onBeforeRefresh.call(_2,_3.pageNumber,_3.pageSize)!=false){
_b(_2,_3.pageNumber);
_3.onRefresh.call(_2,_3.pageNumber,_3.pageSize);
}
});
_4.find("input.pagination-num").unbind(".pagination").bind("keydown.pagination",function(e){
if(e.keyCode==13){
var _9=parseInt($(this).val())||1;
_b(_2,_9);
return false;
}
});
_4.find(".pagination-page-list").unbind(".pagination").bind("change.pagination",function(){
_3.pageSize=$(this).val();
_3.onChangePageSize.call(_2,_3.pageSize);
var _a=Math.ceil(_3.total/_3.pageSize);
_b(_2,_3.pageNumber);
});
};
function _b(_c,_d){
var _e=$.data(_c,"pagination").options;
var _f=Math.ceil(_e.total/_e.pageSize)||1;
var _10=_d;
if(_d<1){
_10=1;
}
if(_d>_f){
_10=_f;
}
_e.pageNumber=_10;
_e.onSelectPage.call(_c,_10,_e.pageSize);
_11(_c);
};
function _11(_12){
var _13=$.data(_12,"pagination").options;
var _14=Math.ceil(_13.total/_13.pageSize)||1;
var num=$(_12).find("input.pagination-num");
num.val(_13.pageNumber);
num.parent().next().find("span").html(_13.afterPageText.replace(/{pages}/,_14));
var _15=_13.displayMsg;
_15=_15.replace(/{from}/,_13.pageSize*(_13.pageNumber-1)+1);
_15=_15.replace(/{to}/,Math.min(_13.pageSize*(_13.pageNumber),_13.total));
_15=_15.replace(/{total}/,_13.total);
$(_12).find(".pagination-info").html(_15);
$("a[icon=pagination-first],a[icon=pagination-prev]",_12).linkbutton({disabled:(_13.pageNumber==1)});
$("a[icon=pagination-next],a[icon=pagination-last]",_12).linkbutton({disabled:(_13.pageNumber==_14)});
if(_13.loading){
$(_12).find("a[icon=pagination-load]").find(".pagination-load").addClass("pagination-loading");
}else{
$(_12).find("a[icon=pagination-load]").find(".pagination-load").removeClass("pagination-loading");
}
};
function _16(_17,_18){
var _19=$.data(_17,"pagination").options;
_19.loading=_18;
if(_19.loading){
$(_17).find("a[icon=pagination-load]").find(".pagination-load").addClass("pagination-loading");
}else{
$(_17).find("a[icon=pagination-load]").find(".pagination-load").removeClass("pagination-loading");
}
};
$.fn.pagination=function(_1a,_1b){
if(typeof _1a=="string"){
return $.fn.pagination.methods[_1a](this,_1b);
}
_1a=_1a||{};
return this.each(function(){
var _1c;
var _1d=$.data(this,"pagination");
if(_1d){
_1c=$.extend(_1d.options,_1a);
}else{
_1c=$.extend({},$.fn.pagination.defaults,_1a);
$.data(this,"pagination",{options:_1c});
}
_1(this);
_11(this);
});
};
$.fn.pagination.methods={options:function(jq){
return $.data(jq[0],"pagination").options;
},loading:function(jq){
return jq.each(function(){
_16(this,true);
});
},loaded:function(jq){
return jq.each(function(){
_16(this,false);
});
}};
$.fn.pagination.defaults={total:1,pageSize:10,pageNumber:1,pageList:[10,20,30,50],loading:false,buttons:null,showPageList:true,showRefresh:true,onSelectPage:function(_1e,_1f){
},onBeforeRefresh:function(_20,_21){
},onRefresh:function(_22,_23){
},onChangePageSize:function(_24){
},beforePageText:"Page",afterPageText:"of {pages}",displayMsg:"Displaying {from} to {to} of {total} items"};
})(jQuery);

