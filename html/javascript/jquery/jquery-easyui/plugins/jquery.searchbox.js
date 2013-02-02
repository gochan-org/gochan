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
$(_2).hide();
var _3=$("<span class=\"searchbox\"></span>").insertAfter(_2);
var _4=$("<input type=\"text\" class=\"searchbox-text\">").appendTo(_3);
$("<span><span class=\"searchbox-button\"></span></span>").appendTo(_3);
var _5=$(_2).attr("name");
if(_5){
_4.attr("name",_5);
$(_2).removeAttr("name").attr("searchboxName",_5);
}
return _3;
};
function _6(_7,_8){
var _9=$.data(_7,"searchbox").options;
var sb=$.data(_7,"searchbox").searchbox;
if(_8){
_9.width=_8;
}
sb.appendTo("body");
if(isNaN(_9.width)){
_9.width=sb.outerWidth();
}
sb._outerWidth(_9.width);
sb.find("input.searchbox-text")._outerWidth(sb.width()-sb.find("a.searchbox-menu").outerWidth()-sb.find("span.searchbox-button").outerWidth());
sb.insertAfter(_7);
};
function _a(_b){
var _c=$.data(_b,"searchbox");
var _d=_c.options;
if(_d.menu){
_c.menu=$(_d.menu).menu({onClick:function(_e){
_f(_e);
}});
var _10=_c.menu.children("div.menu-item:first[selected]");
if(!_10.length){
_10=_c.menu.children("div.menu-item:first");
}
_10.triggerHandler("click");
}else{
_c.searchbox.find("a.searchbox-menu").remove();
_c.menu=null;
}
function _f(_11){
_c.searchbox.find("a.searchbox-menu").remove();
var mb=$("<a class=\"searchbox-menu\" href=\"javascript:void(0)\"></a>").html(_11.text);
mb.prependTo(_c.searchbox).menubutton({menu:_c.menu,iconCls:_11.iconCls});
_c.searchbox.find("input.searchbox-text").attr("name",$(_11.target).attr("name")||_11.text);
_6(_b);
};
};
function _12(_13){
var _14=$.data(_13,"searchbox");
var _15=_14.options;
var _16=_14.searchbox.find("input.searchbox-text");
var _17=_14.searchbox.find(".searchbox-button");
_16.unbind(".searchbox").bind("blur.searchbox",function(e){
_15.value=$(this).val();
if(_15.value==""){
$(this).val(_15.prompt);
$(this).addClass("searchbox-prompt");
}else{
$(this).removeClass("searchbox-prompt");
}
}).bind("focus.searchbox",function(e){
if($(this).val()!=_15.value){
$(this).val(_15.value);
}
$(this).removeClass("searchbox-prompt");
}).bind("keydown.searchbox",function(e){
if(e.keyCode==13){
e.preventDefault();
var _18=$.fn.prop?_16.prop("name"):_16.attr("name");
_15.value=$(this).val();
_15.searcher.call(_13,_15.value,_18);
return false;
}
});
_17.unbind(".searchbox").bind("click.searchbox",function(){
var _19=$.fn.prop?_16.prop("name"):_16.attr("name");
_15.searcher.call(_13,_15.value,_19);
}).bind("mouseenter.searchbox",function(){
$(this).addClass("searchbox-button-hover");
}).bind("mouseleave.searchbox",function(){
$(this).removeClass("searchbox-button-hover");
});
};
function _1a(_1b){
var _1c=$.data(_1b,"searchbox");
var _1d=_1c.options;
var _1e=_1c.searchbox.find("input.searchbox-text");
if(_1d.value==""){
_1e.val(_1d.prompt);
_1e.addClass("searchbox-prompt");
}else{
_1e.val(_1d.value);
_1e.removeClass("searchbox-prompt");
}
};
$.fn.searchbox=function(_1f,_20){
if(typeof _1f=="string"){
return $.fn.searchbox.methods[_1f](this,_20);
}
_1f=_1f||{};
return this.each(function(){
var _21=$.data(this,"searchbox");
if(_21){
$.extend(_21.options,_1f);
}else{
_21=$.data(this,"searchbox",{options:$.extend({},$.fn.searchbox.defaults,$.fn.searchbox.parseOptions(this),_1f),searchbox:_1(this)});
}
_a(this);
_1a(this);
_12(this);
_6(this);
});
};
$.fn.searchbox.methods={options:function(jq){
return $.data(jq[0],"searchbox").options;
},menu:function(jq){
return $.data(jq[0],"searchbox").menu;
},textbox:function(jq){
return $.data(jq[0],"searchbox").searchbox.find("input.searchbox-text");
},getValue:function(jq){
return $.data(jq[0],"searchbox").options.value;
},setValue:function(jq,_22){
return jq.each(function(){
$(this).searchbox("options").value=_22;
$(this).searchbox("textbox").val(_22);
$(this).searchbox("textbox").blur();
});
},getName:function(jq){
return $.data(jq[0],"searchbox").searchbox.find("input.searchbox-text").attr("name");
},selectName:function(jq,_23){
return jq.each(function(){
var _24=$.data(this,"searchbox").menu;
if(_24){
_24.children("div.menu-item[name=\""+_23+"\"]").triggerHandler("click");
}
});
},destroy:function(jq){
return jq.each(function(){
var _25=$(this).searchbox("menu");
if(_25){
_25.menu("destroy");
}
$.data(this,"searchbox").searchbox.remove();
$(this).remove();
});
},resize:function(jq,_26){
return jq.each(function(){
_6(this,_26);
});
}};
$.fn.searchbox.parseOptions=function(_27){
var t=$(_27);
return {width:(parseInt(_27.style.width)||undefined),prompt:t.attr("prompt"),value:t.val(),menu:t.attr("menu"),searcher:(t.attr("searcher")?eval(t.attr("searcher")):undefined)};
};
$.fn.searchbox.defaults={width:"auto",prompt:"",value:"",menu:null,searcher:function(_28,_29){
}};
})(jQuery);

