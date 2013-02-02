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
$(_2).addClass("progressbar");
$(_2).html("<div class=\"progressbar-text\"></div><div class=\"progressbar-value\">&nbsp;</div>");
return $(_2);
};
function _3(_4,_5){
var _6=$.data(_4,"progressbar").options;
var _7=$.data(_4,"progressbar").bar;
if(_5){
_6.width=_5;
}
_7._outerWidth(_6.width);
_7.find("div.progressbar-text").width(_7.width());
};
$.fn.progressbar=function(_8,_9){
if(typeof _8=="string"){
var _a=$.fn.progressbar.methods[_8];
if(_a){
return _a(this,_9);
}
}
_8=_8||{};
return this.each(function(){
var _b=$.data(this,"progressbar");
if(_b){
$.extend(_b.options,_8);
}else{
_b=$.data(this,"progressbar",{options:$.extend({},$.fn.progressbar.defaults,$.fn.progressbar.parseOptions(this),_8),bar:_1(this)});
}
$(this).progressbar("setValue",_b.options.value);
_3(this);
});
};
$.fn._outerWidth=function(_c){
return this.each(function(){
if(!$.boxModel&&$.browser.msie){
$(this).width(_c);
}else{
$(this).width(_c-($(this).outerWidth()-$(this).width()));
}
});
};
$.fn.progressbar.methods={options:function(jq){
return $.data(jq[0],"progressbar").options;
},resize:function(jq,_d){
return jq.each(function(){
_3(this,_d);
});
},getValue:function(jq){
return $.data(jq[0],"progressbar").options.value;
},setValue:function(jq,_e){
if(_e<0){
_e=0;
}
if(_e>100){
_e=100;
}
return jq.each(function(){
var _f=$.data(this,"progressbar").options;
var _10=_f.text.replace(/{value}/,_e);
var _11=_f.value;
_f.value=_e;
$(this).find("div.progressbar-value").width(_e+"%");
$(this).find("div.progressbar-text").html(_10);
if(_11!=_e){
_f.onChange.call(this,_e,_11);
}
});
}};
$.fn.progressbar.parseOptions=function(_12){
var t=$(_12);
return {width:(parseInt(_12.style.width)||undefined),value:(t.attr("value")?parseInt(t.attr("value")):undefined),text:t.attr("text")};
};
$.fn.progressbar.defaults={width:"auto",value:0,text:"{value}%",onChange:function(_13,_14){
}};
})(jQuery);

