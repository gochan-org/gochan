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
var _3=$(_2).children("div.tabs-header");
var _4=0;
$("ul.tabs li",_3).each(function(){
_4+=$(this).outerWidth(true);
});
var _5=_3.children("div.tabs-wrap").width();
var _6=parseInt(_3.find("ul.tabs").css("padding-left"));
return _4-_5+_6;
};
function _7(_8){
var _9=$.data(_8,"tabs").options;
var _a=$(_8).children("div.tabs-header");
var _b=_a.children("div.tabs-tool");
var _c=_a.children("div.tabs-scroller-left");
var _d=_a.children("div.tabs-scroller-right");
var _e=_a.children("div.tabs-wrap");
_b._outerHeight(_a.outerHeight()-(_9.plain?2:0));
var _f=0;
$("ul.tabs li",_a).each(function(){
_f+=$(this).outerWidth(true);
});
var _10=_a.width()-_b.outerWidth();
if(_f>_10){
_c.show();
_d.show();
_b.css("right",_d.outerWidth());
_e.css({marginLeft:_c.outerWidth(),marginRight:_d.outerWidth()+_b.outerWidth(),left:0,width:_10-_c.outerWidth()-_d.outerWidth()});
}else{
_c.hide();
_d.hide();
_b.css("right",0);
_e.css({marginLeft:0,marginRight:_b.outerWidth(),left:0,width:_10});
_e.scrollLeft(0);
}
};
function _11(_12){
var _13=$.data(_12,"tabs").options;
var _14=$(_12).children("div.tabs-header");
if(_13.tools){
if(typeof _13.tools=="string"){
$(_13.tools).addClass("tabs-tool").appendTo(_14);
$(_13.tools).show();
}else{
_14.children("div.tabs-tool").remove();
var _15=$("<div class=\"tabs-tool\"></div>").appendTo(_14);
for(var i=0;i<_13.tools.length;i++){
var _16=$("<a href=\"javascript:void(0);\"></a>").appendTo(_15);
_16[0].onclick=eval(_13.tools[i].handler||function(){
});
_16.linkbutton($.extend({},_13.tools[i],{plain:true}));
}
}
}else{
_14.children("div.tabs-tool").remove();
}
};
function _17(_18){
var _19=$.data(_18,"tabs").options;
var cc=$(_18);
if(_19.fit==true){
var p=cc.parent();
p.addClass("panel-noscroll");
_19.width=p.width();
_19.height=p.height();
}
cc.width(_19.width).height(_19.height);
var _1a=$(_18).children("div.tabs-header");
_1a._outerWidth(_19.width);
_7(_18);
var _1b=$(_18).children("div.tabs-panels");
var _1c=_19.height;
if(!isNaN(_1c)){
_1b._outerHeight(_1c-_1a.outerHeight());
}else{
_1b.height("auto");
}
var _1d=_19.width;
if(!isNaN(_1d)){
_1b._outerWidth(_1d);
}else{
_1b.width("auto");
}
};
function _1e(_1f){
var _20=$.data(_1f,"tabs").options;
var tab=_21(_1f);
if(tab){
var _22=$(_1f).children("div.tabs-panels");
var _23=_20.width=="auto"?"auto":_22.width();
var _24=_20.height=="auto"?"auto":_22.height();
tab.panel("resize",{width:_23,height:_24});
}
};
function _25(_26){
var cc=$(_26);
cc.addClass("tabs-container");
cc.wrapInner("<div class=\"tabs-panels\"/>");
$("<div class=\"tabs-header\">"+"<div class=\"tabs-scroller-left\"></div>"+"<div class=\"tabs-scroller-right\"></div>"+"<div class=\"tabs-wrap\">"+"<ul class=\"tabs\"></ul>"+"</div>"+"</div>").prependTo(_26);
var _27=[];
var tp=cc.children("div.tabs-panels");
tp.children("div[selected]").attr("toselect","true");
tp.children("div").each(function(){
var pp=$(this);
_27.push(pp);
_35(_26,pp);
});
cc.children("div.tabs-header").find(".tabs-scroller-left, .tabs-scroller-right").hover(function(){
$(this).addClass("tabs-scroller-over");
},function(){
$(this).removeClass("tabs-scroller-over");
});
cc.bind("_resize",function(e,_28){
var _29=$.data(_26,"tabs").options;
if(_29.fit==true||_28){
_17(_26);
_1e(_26);
}
return false;
});
return _27;
};
function _2a(_2b){
var _2c=$.data(_2b,"tabs").options;
var _2d=$(_2b).children("div.tabs-header");
var _2e=$(_2b).children("div.tabs-panels");
if(_2c.plain==true){
_2d.addClass("tabs-header-plain");
}else{
_2d.removeClass("tabs-header-plain");
}
if(_2c.border==true){
_2d.removeClass("tabs-header-noborder");
_2e.removeClass("tabs-panels-noborder");
}else{
_2d.addClass("tabs-header-noborder");
_2e.addClass("tabs-panels-noborder");
}
$(".tabs-scroller-left",_2d).unbind(".tabs").bind("click.tabs",function(){
var _2f=$(".tabs-wrap",_2d);
var pos=_2f.scrollLeft()-_2c.scrollIncrement;
_2f.animate({scrollLeft:pos},_2c.scrollDuration);
});
$(".tabs-scroller-right",_2d).unbind(".tabs").bind("click.tabs",function(){
var _30=$(".tabs-wrap",_2d);
var pos=Math.min(_30.scrollLeft()+_2c.scrollIncrement,_1(_2b));
_30.animate({scrollLeft:pos},_2c.scrollDuration);
});
var _31=$.data(_2b,"tabs").tabs;
for(var i=0,len=_31.length;i<len;i++){
var _32=_31[i];
var tab=_32.panel("options").tab;
tab.unbind(".tabs").bind("click.tabs",{p:_32},function(e){
_44(_2b,_34(_2b,e.data.p));
}).bind("contextmenu.tabs",{p:_32},function(e){
_2c.onContextMenu.call(_2b,e,e.data.p.panel("options").title);
});
tab.find("a.tabs-close").unbind(".tabs").bind("click.tabs",{p:_32},function(e){
_33(_2b,_34(_2b,e.data.p));
return false;
});
}
};
function _35(_36,pp,_37){
_37=_37||{};
pp.panel($.extend({},_37,{border:false,noheader:true,closed:true,doSize:false,iconCls:(_37.icon?_37.icon:undefined),onLoad:function(){
if(_37.onLoad){
_37.onLoad.call(this,arguments);
}
$.data(_36,"tabs").options.onLoad.call(_36,pp);
}}));
var _38=pp.panel("options");
var _39=$(_36).children("div.tabs-header");
var _3a=$("ul.tabs",_39);
var tab=$("<li></li>").appendTo(_3a);
var _3b=$("<a href=\"javascript:void(0)\" class=\"tabs-inner\"></a>").appendTo(tab);
var _3c=$("<span class=\"tabs-title\"></span>").html(_38.title).appendTo(_3b);
var _3d=$("<span class=\"tabs-icon\"></span>").appendTo(_3b);
if(_38.closable){
_3c.addClass("tabs-closable");
$("<a href=\"javascript:void(0)\" class=\"tabs-close\"></a>").appendTo(tab);
}
if(_38.iconCls){
_3c.addClass("tabs-with-icon");
_3d.addClass(_38.iconCls);
}
if(_38.tools){
var _3e=$("<span class=\"tabs-p-tool\"></span>").insertAfter(_3b);
if(typeof _38.tools=="string"){
$(_38.tools).children().appendTo(_3e);
}else{
for(var i=0;i<_38.tools.length;i++){
var t=$("<a href=\"javascript:void(0)\"></a>").appendTo(_3e);
t.addClass(_38.tools[i].iconCls);
if(_38.tools[i].handler){
t.bind("click",eval(_38.tools[i].handler));
}
}
}
var pr=_3e.children().length*12;
if(_38.closable){
pr+=8;
}else{
pr-=3;
_3e.css("right","5px");
}
_3c.css("padding-right",pr+"px");
}
_38.tab=tab;
};
function _3f(_40,_41){
var _42=$.data(_40,"tabs").options;
var _43=$.data(_40,"tabs").tabs;
if(_41.selected==undefined){
_41.selected=true;
}
var pp=$("<div></div>").appendTo($(_40).children("div.tabs-panels"));
_43.push(pp);
_35(_40,pp,_41);
_42.onAdd.call(_40,_41.title);
_7(_40);
_2a(_40);
if(_41.selected){
_44(_40,_43.length-1);
}
};
function _45(_46,_47){
var _48=$.data(_46,"tabs").selectHis;
var pp=_47.tab;
var _49=pp.panel("options").title;
pp.panel($.extend({},_47.options,{iconCls:(_47.options.icon?_47.options.icon:undefined)}));
var _4a=pp.panel("options");
var tab=_4a.tab;
tab.find("span.tabs-icon").attr("class","tabs-icon");
tab.find("a.tabs-close").remove();
tab.find("span.tabs-title").html(_4a.title);
if(_4a.closable){
tab.find("span.tabs-title").addClass("tabs-closable");
$("<a href=\"javascript:void(0)\" class=\"tabs-close\"></a>").appendTo(tab);
}else{
tab.find("span.tabs-title").removeClass("tabs-closable");
}
if(_4a.iconCls){
tab.find("span.tabs-title").addClass("tabs-with-icon");
tab.find("span.tabs-icon").addClass(_4a.iconCls);
}else{
tab.find("span.tabs-title").removeClass("tabs-with-icon");
}
if(_49!=_4a.title){
for(var i=0;i<_48.length;i++){
if(_48[i]==_49){
_48[i]=_4a.title;
}
}
}
_2a(_46);
$.data(_46,"tabs").options.onUpdate.call(_46,_4a.title);
};
function _33(_4b,_4c){
var _4d=$.data(_4b,"tabs").options;
var _4e=$.data(_4b,"tabs").tabs;
var _4f=$.data(_4b,"tabs").selectHis;
if(!_50(_4b,_4c)){
return;
}
var tab=_51(_4b,_4c);
var _52=tab.panel("options").title;
if(_4d.onBeforeClose.call(_4b,_52)==false){
return;
}
var tab=_51(_4b,_4c,true);
tab.panel("options").tab.remove();
tab.panel("destroy");
_4d.onClose.call(_4b,_52);
_7(_4b);
for(var i=0;i<_4f.length;i++){
if(_4f[i]==_52){
_4f.splice(i,1);
i--;
}
}
var _53=_4f.pop();
if(_53){
_44(_4b,_53);
}else{
if(_4e.length){
_44(_4b,0);
}
}
};
function _51(_54,_55,_56){
var _57=$.data(_54,"tabs").tabs;
if(typeof _55=="number"){
if(_55<0||_55>=_57.length){
return null;
}else{
var tab=_57[_55];
if(_56){
_57.splice(_55,1);
}
return tab;
}
}
for(var i=0;i<_57.length;i++){
var tab=_57[i];
if(tab.panel("options").title==_55){
if(_56){
_57.splice(i,1);
}
return tab;
}
}
return null;
};
function _34(_58,tab){
var _59=$.data(_58,"tabs").tabs;
for(var i=0;i<_59.length;i++){
if(_59[i][0]==$(tab)[0]){
return i;
}
}
return -1;
};
function _21(_5a){
var _5b=$.data(_5a,"tabs").tabs;
for(var i=0;i<_5b.length;i++){
var tab=_5b[i];
if(tab.panel("options").closed==false){
return tab;
}
}
return null;
};
function _5c(_5d){
var _5e=$.data(_5d,"tabs").tabs;
for(var i=0;i<_5e.length;i++){
if(_5e[i].attr("toselect")=="true"){
_44(_5d,i);
return;
}
}
if(_5e.length){
_44(_5d,0);
}
};
function _44(_5f,_60){
var _61=$.data(_5f,"tabs").options;
var _62=$.data(_5f,"tabs").tabs;
var _63=$.data(_5f,"tabs").selectHis;
if(_62.length==0){
return;
}
var _64=_51(_5f,_60);
if(!_64){
return;
}
var _65=_21(_5f);
if(_65){
_65.panel("close");
_65.panel("options").tab.removeClass("tabs-selected");
}
_64.panel("open");
var _66=_64.panel("options").title;
_63.push(_66);
var tab=_64.panel("options").tab;
tab.addClass("tabs-selected");
var _67=$(_5f).find(">div.tabs-header div.tabs-wrap");
var _68=tab.position().left+_67.scrollLeft();
var _69=_68-_67.scrollLeft();
var _6a=_69+tab.outerWidth();
if(_69<0||_6a>_67.innerWidth()){
var pos=Math.min(_68-(_67.width()-tab.width())/2,_1(_5f));
_67.animate({scrollLeft:pos},_61.scrollDuration);
}else{
var pos=Math.min(_67.scrollLeft(),_1(_5f));
_67.animate({scrollLeft:pos},_61.scrollDuration);
}
_1e(_5f);
_61.onSelect.call(_5f,_66);
};
function _50(_6b,_6c){
return _51(_6b,_6c)!=null;
};
$.fn.tabs=function(_6d,_6e){
if(typeof _6d=="string"){
return $.fn.tabs.methods[_6d](this,_6e);
}
_6d=_6d||{};
return this.each(function(){
var _6f=$.data(this,"tabs");
var _70;
if(_6f){
_70=$.extend(_6f.options,_6d);
_6f.options=_70;
}else{
$.data(this,"tabs",{options:$.extend({},$.fn.tabs.defaults,$.fn.tabs.parseOptions(this),_6d),tabs:_25(this),selectHis:[]});
}
_11(this);
_2a(this);
_17(this);
_5c(this);
});
};
$.fn.tabs.methods={options:function(jq){
return $.data(jq[0],"tabs").options;
},tabs:function(jq){
return $.data(jq[0],"tabs").tabs;
},resize:function(jq){
return jq.each(function(){
_17(this);
_1e(this);
});
},add:function(jq,_71){
return jq.each(function(){
_3f(this,_71);
});
},close:function(jq,_72){
return jq.each(function(){
_33(this,_72);
});
},getTab:function(jq,_73){
return _51(jq[0],_73);
},getTabIndex:function(jq,tab){
return _34(jq[0],tab);
},getSelected:function(jq){
return _21(jq[0]);
},select:function(jq,_74){
return jq.each(function(){
_44(this,_74);
});
},exists:function(jq,_75){
return _50(jq[0],_75);
},update:function(jq,_76){
return jq.each(function(){
_45(this,_76);
});
}};
$.fn.tabs.parseOptions=function(_77){
var t=$(_77);
return {width:(parseInt(_77.style.width)||undefined),height:(parseInt(_77.style.height)||undefined),fit:(t.attr("fit")?t.attr("fit")=="true":undefined),border:(t.attr("border")?t.attr("border")=="true":undefined),plain:(t.attr("plain")?t.attr("plain")=="true":undefined),tools:t.attr("tools")};
};
$.fn.tabs.defaults={width:"auto",height:"auto",plain:false,fit:false,border:true,tools:null,scrollIncrement:100,scrollDuration:400,onLoad:function(_78){
},onSelect:function(_79){
},onBeforeClose:function(_7a){
},onClose:function(_7b){
},onAdd:function(_7c){
},onUpdate:function(_7d){
},onContextMenu:function(e,_7e){
}};
})(jQuery);

