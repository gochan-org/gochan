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
$(_2).appendTo("body");
$(_2).addClass("menu-top");
var _3=[];
_4($(_2));
var _5=null;
for(var i=0;i<_3.length;i++){
var _6=_3[i];
_7(_6);
_6.children("div.menu-item").each(function(){
_f(_2,$(this));
});
_6.bind("mouseenter",function(){
if(_5){
clearTimeout(_5);
_5=null;
}
}).bind("mouseleave",function(){
_5=setTimeout(function(){
_18(_2);
},100);
});
}
function _4(_8){
_3.push(_8);
_8.find(">div").each(function(){
var _9=$(this);
var _a=_9.find(">div");
if(_a.length){
_a.insertAfter(_2);
_9[0].submenu=_a;
_4(_a);
}
});
};
function _7(_b){
_b.addClass("menu").find(">div").each(function(){
var _c=$(this);
if(_c.hasClass("menu-sep")){
_c.html("&nbsp;");
}else{
var _d=_c.addClass("menu-item").html();
_c.empty().append($("<div class=\"menu-text\"></div>").html(_d));
var _e=_c.attr("iconCls")||_c.attr("icon");
if(_e){
$("<div class=\"menu-icon\"></div>").addClass(_e).appendTo(_c);
}
if(_c[0].submenu){
$("<div class=\"menu-rightarrow\"></div>").appendTo(_c);
}
_c._outerHeight(22);
}
});
_b.hide();
};
};
function _f(_10,_11){
_11.unbind(".menu");
_11.bind("mousedown.menu",function(){
return false;
}).bind("click.menu",function(){
if($(this).hasClass("menu-item-disabled")){
return;
}
if(!this.submenu){
_18(_10);
var _12=$(this).attr("href");
if(_12){
location.href=_12;
}
}
var _13=$(_10).menu("getItem",this);
$.data(_10,"menu").options.onClick.call(_10,_13);
}).bind("mouseenter.menu",function(e){
_11.siblings().each(function(){
if(this.submenu){
_17(this.submenu);
}
$(this).removeClass("menu-active");
});
_11.addClass("menu-active");
if($(this).hasClass("menu-item-disabled")){
_11.addClass("menu-active-disabled");
return;
}
var _14=_11[0].submenu;
if(_14){
var _15=_11.offset().left+_11.outerWidth()-2;
if(_15+_14.outerWidth()+5>$(window).width()+$(document).scrollLeft()){
_15=_11.offset().left-_14.outerWidth()+2;
}
var top=_11.offset().top-3;
if(top+_14.outerHeight()>$(window).height()+$(document).scrollTop()){
top=$(window).height()+$(document).scrollTop()-_14.outerHeight()-5;
}
_1e(_14,{left:_15,top:top});
}
}).bind("mouseleave.menu",function(e){
_11.removeClass("menu-active menu-active-disabled");
var _16=_11[0].submenu;
if(_16){
if(e.pageX>=parseInt(_16.css("left"))){
_11.addClass("menu-active");
}else{
_17(_16);
}
}else{
_11.removeClass("menu-active");
}
});
};
function _18(_19){
var _1a=$.data(_19,"menu").options;
_17($(_19));
$(document).unbind(".menu");
_1a.onHide.call(_19);
return false;
};
function _1b(_1c,pos){
var _1d=$.data(_1c,"menu").options;
if(pos){
_1d.left=pos.left;
_1d.top=pos.top;
if(_1d.left+$(_1c).outerWidth()>$(window).width()+$(document).scrollLeft()){
_1d.left=$(window).width()+$(document).scrollLeft()-$(_1c).outerWidth()-5;
}
if(_1d.top+$(_1c).outerHeight()>$(window).height()+$(document).scrollTop()){
_1d.top-=$(_1c).outerHeight();
}
}
_1e($(_1c),{left:_1d.left,top:_1d.top},function(){
$(document).unbind(".menu").bind("mousedown.menu",function(){
_18(_1c);
$(document).unbind(".menu");
return false;
});
_1d.onShow.call(_1c);
});
};
function _1e(_1f,pos,_20){
if(!_1f){
return;
}
if(pos){
_1f.css(pos);
}
_1f.show(0,function(){
if(!_1f[0].shadow){
_1f[0].shadow=$("<div class=\"menu-shadow\"></div>").insertAfter(_1f);
}
_1f[0].shadow.css({display:"block",zIndex:$.fn.menu.defaults.zIndex++,left:_1f.css("left"),top:_1f.css("top"),width:_1f.outerWidth(),height:_1f.outerHeight()});
_1f.css("z-index",$.fn.menu.defaults.zIndex++);
if(_20){
_20();
}
});
};
function _17(_21){
if(!_21){
return;
}
_22(_21);
_21.find("div.menu-item").each(function(){
if(this.submenu){
_17(this.submenu);
}
$(this).removeClass("menu-active");
});
function _22(m){
m.stop(true,true);
if(m[0].shadow){
m[0].shadow.hide();
}
m.hide();
};
};
function _23(_24,_25){
var _26=null;
var tmp=$("<div></div>");
function _27(_28){
_28.children("div.menu-item").each(function(){
var _29=$(_24).menu("getItem",this);
var s=tmp.empty().html(_29.text).text();
if(_25==$.trim(s)){
_26=_29;
}else{
if(this.submenu&&!_26){
_27(this.submenu);
}
}
});
};
_27($(_24));
tmp.remove();
return _26;
};
function _2a(_2b,_2c,_2d){
var t=$(_2c);
if(_2d){
t.addClass("menu-item-disabled");
if(_2c.onclick){
_2c.onclick1=_2c.onclick;
_2c.onclick=null;
}
}else{
t.removeClass("menu-item-disabled");
if(_2c.onclick1){
_2c.onclick=_2c.onclick1;
_2c.onclick1=null;
}
}
};
function _2e(_2f,_30){
var _31=$(_2f);
if(_30.parent){
_31=_30.parent.submenu;
}
var _32=$("<div class=\"menu-item\"></div>").appendTo(_31);
$("<div class=\"menu-text\"></div>").html(_30.text).appendTo(_32);
if(_30.iconCls){
$("<div class=\"menu-icon\"></div>").addClass(_30.iconCls).appendTo(_32);
}
if(_30.id){
_32.attr("id",_30.id);
}
if(_30.href){
_32.attr("href",_30.href);
}
if(_30.onclick){
if(typeof _30.onclick=="string"){
_32.attr("onclick",_30.onclick);
}else{
_32[0].onclick=eval(_30.onclick);
}
}
if(_30.handler){
_32[0].onclick=eval(_30.handler);
}
_f(_2f,_32);
};
function _33(_34,_35){
function _36(el){
if(el.submenu){
el.submenu.children("div.menu-item").each(function(){
_36(this);
});
var _37=el.submenu[0].shadow;
if(_37){
_37.remove();
}
el.submenu.remove();
}
$(el).remove();
};
_36(_35);
};
function _38(_39){
$(_39).children("div.menu-item").each(function(){
_33(_39,this);
});
if(_39.shadow){
_39.shadow.remove();
}
$(_39).remove();
};
$.fn.menu=function(_3a,_3b){
if(typeof _3a=="string"){
return $.fn.menu.methods[_3a](this,_3b);
}
_3a=_3a||{};
return this.each(function(){
var _3c=$.data(this,"menu");
if(_3c){
$.extend(_3c.options,_3a);
}else{
_3c=$.data(this,"menu",{options:$.extend({},$.fn.menu.defaults,_3a)});
_1(this);
}
$(this).css({left:_3c.options.left,top:_3c.options.top});
});
};
$.fn._outerWidth=function(_3d){
return this.each(function(){
if(!$.boxModel&&$.browser.msie){
$(this).width(_3d);
}else{
$(this).width(_3d-($(this).outerWidth()-$(this).width()));
}
});
};
$.fn._outerHeight=function(_3e){
return this.each(function(){
if(!$.boxModel&&$.browser.msie){
$(this).height(_3e);
}else{
$(this).height(_3e-($(this).outerHeight()-$(this).height()));
}
});
};
$.fn.menu.methods={show:function(jq,pos){
return jq.each(function(){
_1b(this,pos);
});
},hide:function(jq){
return jq.each(function(){
_18(this);
});
},destroy:function(jq){
return jq.each(function(){
_38(this);
});
},setText:function(jq,_3f){
return jq.each(function(){
$(_3f.target).children("div.menu-text").html(_3f.text);
});
},setIcon:function(jq,_40){
return jq.each(function(){
var _41=$(this).menu("getItem",_40.target);
if(_41.iconCls){
$(_41.target).children("div.menu-icon").removeClass(_41.iconCls).addClass(_40.iconCls);
}else{
$("<div class=\"menu-icon\"></div>").addClass(_40.iconCls).appendTo(_40.target);
}
});
},getItem:function(jq,_42){
var _43={target:_42,id:$(_42).attr("id"),text:$.trim($(_42).children("div.menu-text").html()),disabled:$(_42).hasClass("menu-item-disabled"),href:$(_42).attr("href"),onclick:_42.onclick};
var _44=$(_42).children("div.menu-icon");
if(_44.length){
var cc=[];
var aa=_44.attr("class").split(" ");
for(var i=0;i<aa.length;i++){
if(aa[i]!="menu-icon"){
cc.push(aa[i]);
}
}
_43.iconCls=cc.join(" ");
}
return _43;
},findItem:function(jq,_45){
return _23(jq[0],_45);
},appendItem:function(jq,_46){
return jq.each(function(){
_2e(this,_46);
});
},removeItem:function(jq,_47){
return jq.each(function(){
_33(this,_47);
});
},enableItem:function(jq,_48){
return jq.each(function(){
_2a(this,_48,false);
});
},disableItem:function(jq,_49){
return jq.each(function(){
_2a(this,_49,true);
});
}};
$.fn.menu.defaults={zIndex:110000,left:0,top:0,onShow:function(){
},onHide:function(){
},onClick:function(_4a){
}};
})(jQuery);

