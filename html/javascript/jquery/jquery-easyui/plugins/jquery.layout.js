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
var _1=false;
function _2(_3){
var _4=$.data(_3,"layout").options;
var _5=$.data(_3,"layout").panels;
var cc=$(_3);
if(_4.fit==true){
var p=cc.parent();
p.addClass("panel-noscroll");
cc.width(p.width());
cc.height(p.height());
}
var _6={top:0,left:0,width:cc.width(),height:cc.height()};
function _7(pp){
if(pp.length==0){
return;
}
pp.panel("resize",{width:cc.width(),height:pp.panel("options").height,left:0,top:0});
_6.top+=pp.panel("options").height;
_6.height-=pp.panel("options").height;
};
if(_b(_5.expandNorth)){
_7(_5.expandNorth);
}else{
_7(_5.north);
}
function _8(pp){
if(pp.length==0){
return;
}
pp.panel("resize",{width:cc.width(),height:pp.panel("options").height,left:0,top:cc.height()-pp.panel("options").height});
_6.height-=pp.panel("options").height;
};
if(_b(_5.expandSouth)){
_8(_5.expandSouth);
}else{
_8(_5.south);
}
function _9(pp){
if(pp.length==0){
return;
}
pp.panel("resize",{width:pp.panel("options").width,height:_6.height,left:cc.width()-pp.panel("options").width,top:_6.top});
_6.width-=pp.panel("options").width;
};
if(_b(_5.expandEast)){
_9(_5.expandEast);
}else{
_9(_5.east);
}
function _a(pp){
if(pp.length==0){
return;
}
pp.panel("resize",{width:pp.panel("options").width,height:_6.height,left:0,top:_6.top});
_6.left+=pp.panel("options").width;
_6.width-=pp.panel("options").width;
};
if(_b(_5.expandWest)){
_a(_5.expandWest);
}else{
_a(_5.west);
}
_5.center.panel("resize",_6);
};
function _c(_d){
var cc=$(_d);
if(cc[0].tagName=="BODY"){
$("html").css({height:"100%",overflow:"hidden"});
$("body").css({height:"100%",overflow:"hidden",border:"none"});
}
cc.addClass("layout");
cc.css({margin:0,padding:0});
$("<div class=\"layout-split-proxy-h\"></div>").appendTo(cc);
$("<div class=\"layout-split-proxy-v\"></div>").appendTo(cc);
cc.children("div[region]").each(function(){
var _e=$(this).attr("region");
_11(_d,{region:_e});
});
cc.bind("_resize",function(e,_f){
var _10=$.data(_d,"layout").options;
if(_10.fit==true||_f){
_2(_d);
}
return false;
});
};
function _11(_12,_13){
_13.region=_13.region||"center";
var _14=$.data(_12,"layout").panels;
var cc=$(_12);
var dir=_13.region;
if(_14[dir].length){
return;
}
var pp=cc.children("div[region="+dir+"]");
if(!pp.length){
pp=$("<div></div>").appendTo(cc);
}
pp.panel($.extend({},{width:(pp.length?parseInt(pp[0].style.width)||pp.outerWidth():"auto"),height:(pp.length?parseInt(pp[0].style.height)||pp.outerHeight():"auto"),split:(pp.attr("split")?pp.attr("split")=="true":undefined),doSize:false,cls:("layout-panel layout-panel-"+dir),bodyCls:"layout-body",onOpen:function(){
var _15={north:"up",south:"down",east:"right",west:"left"};
if(!_15[dir]){
return;
}
var _16="layout-button-"+_15[dir];
var _17=$(this).panel("header").children("div.panel-tool");
if(!_17.children("a."+_16).length){
var t=$("<a href=\"javascript:void(0)\"></a>").addClass(_16).appendTo(_17);
t.bind("click",{dir:dir},function(e){
_25(_12,e.data.dir);
return false;
});
}
}},_13));
_14[dir]=pp;
if(pp.panel("options").split){
var _18=pp.panel("panel");
_18.addClass("layout-split-"+dir);
var _19="";
if(dir=="north"){
_19="s";
}
if(dir=="south"){
_19="n";
}
if(dir=="east"){
_19="w";
}
if(dir=="west"){
_19="e";
}
_18.resizable({handles:_19,onStartResize:function(e){
_1=true;
if(dir=="north"||dir=="south"){
var _1a=$(">div.layout-split-proxy-v",_12);
}else{
var _1a=$(">div.layout-split-proxy-h",_12);
}
var top=0,_1b=0,_1c=0,_1d=0;
var pos={display:"block"};
if(dir=="north"){
pos.top=parseInt(_18.css("top"))+_18.outerHeight()-_1a.height();
pos.left=parseInt(_18.css("left"));
pos.width=_18.outerWidth();
pos.height=_1a.height();
}else{
if(dir=="south"){
pos.top=parseInt(_18.css("top"));
pos.left=parseInt(_18.css("left"));
pos.width=_18.outerWidth();
pos.height=_1a.height();
}else{
if(dir=="east"){
pos.top=parseInt(_18.css("top"))||0;
pos.left=parseInt(_18.css("left"))||0;
pos.width=_1a.width();
pos.height=_18.outerHeight();
}else{
if(dir=="west"){
pos.top=parseInt(_18.css("top"))||0;
pos.left=_18.outerWidth()-_1a.width();
pos.width=_1a.width();
pos.height=_18.outerHeight();
}
}
}
}
_1a.css(pos);
$("<div class=\"layout-mask\"></div>").css({left:0,top:0,width:cc.width(),height:cc.height()}).appendTo(cc);
},onResize:function(e){
if(dir=="north"||dir=="south"){
var _1e=$(">div.layout-split-proxy-v",_12);
_1e.css("top",e.pageY-$(_12).offset().top-_1e.height()/2);
}else{
var _1e=$(">div.layout-split-proxy-h",_12);
_1e.css("left",e.pageX-$(_12).offset().left-_1e.width()/2);
}
return false;
},onStopResize:function(){
$(">div.layout-split-proxy-v",_12).css("display","none");
$(">div.layout-split-proxy-h",_12).css("display","none");
var _1f=pp.panel("options");
_1f.width=_18.outerWidth();
_1f.height=_18.outerHeight();
_1f.left=_18.css("left");
_1f.top=_18.css("top");
pp.panel("resize");
_2(_12);
_1=false;
cc.find(">div.layout-mask").remove();
}});
}
};
function _20(_21,_22){
var _23=$.data(_21,"layout").panels;
if(_23[_22].length){
_23[_22].panel("destroy");
_23[_22]=$();
var _24="expand"+_22.substring(0,1).toUpperCase()+_22.substring(1);
if(_23[_24]){
_23[_24].panel("destroy");
_23[_24]=undefined;
}
}
};
function _25(_26,_27,_28){
if(_28==undefined){
_28="normal";
}
var _29=$.data(_26,"layout").panels;
var p=_29[_27];
if(p.panel("options").onBeforeCollapse.call(p)==false){
return;
}
var _2a="expand"+_27.substring(0,1).toUpperCase()+_27.substring(1);
if(!_29[_2a]){
_29[_2a]=_2b(_27);
_29[_2a].panel("panel").click(function(){
var _2c=_2d();
p.panel("expand",false).panel("open").panel("resize",_2c.collapse);
p.panel("panel").animate(_2c.expand);
return false;
});
}
var _2e=_2d();
if(!_b(_29[_2a])){
_29.center.panel("resize",_2e.resizeC);
}
p.panel("panel").animate(_2e.collapse,_28,function(){
p.panel("collapse",false).panel("close");
_29[_2a].panel("open").panel("resize",_2e.expandP);
});
function _2b(dir){
var _2f;
if(dir=="east"){
_2f="layout-button-left";
}else{
if(dir=="west"){
_2f="layout-button-right";
}else{
if(dir=="north"){
_2f="layout-button-down";
}else{
if(dir=="south"){
_2f="layout-button-up";
}
}
}
}
var p=$("<div></div>").appendTo(_26).panel({cls:"layout-expand",title:"&nbsp;",closed:true,doSize:false,tools:[{iconCls:_2f,handler:function(){
_30(_26,_27);
return false;
}}]});
p.panel("panel").hover(function(){
$(this).addClass("layout-expand-over");
},function(){
$(this).removeClass("layout-expand-over");
});
return p;
};
function _2d(){
var cc=$(_26);
if(_27=="east"){
return {resizeC:{width:_29.center.panel("options").width+_29["east"].panel("options").width-28},expand:{left:cc.width()-_29["east"].panel("options").width},expandP:{top:_29["east"].panel("options").top,left:cc.width()-28,width:28,height:_29["center"].panel("options").height},collapse:{left:cc.width()}};
}else{
if(_27=="west"){
return {resizeC:{width:_29.center.panel("options").width+_29["west"].panel("options").width-28,left:28},expand:{left:0},expandP:{left:0,top:_29["west"].panel("options").top,width:28,height:_29["center"].panel("options").height},collapse:{left:-_29["west"].panel("options").width}};
}else{
if(_27=="north"){
var hh=cc.height()-28;
if(_b(_29.expandSouth)){
hh-=_29.expandSouth.panel("options").height;
}else{
if(_b(_29.south)){
hh-=_29.south.panel("options").height;
}
}
_29.east.panel("resize",{top:28,height:hh});
_29.west.panel("resize",{top:28,height:hh});
if(_b(_29.expandEast)){
_29.expandEast.panel("resize",{top:28,height:hh});
}
if(_b(_29.expandWest)){
_29.expandWest.panel("resize",{top:28,height:hh});
}
return {resizeC:{top:28,height:hh},expand:{top:0},expandP:{top:0,left:0,width:cc.width(),height:28},collapse:{top:-_29["north"].panel("options").height}};
}else{
if(_27=="south"){
var hh=cc.height()-28;
if(_b(_29.expandNorth)){
hh-=_29.expandNorth.panel("options").height;
}else{
if(_b(_29.north)){
hh-=_29.north.panel("options").height;
}
}
_29.east.panel("resize",{height:hh});
_29.west.panel("resize",{height:hh});
if(_b(_29.expandEast)){
_29.expandEast.panel("resize",{height:hh});
}
if(_b(_29.expandWest)){
_29.expandWest.panel("resize",{height:hh});
}
return {resizeC:{height:hh},expand:{top:cc.height()-_29["south"].panel("options").height},expandP:{top:cc.height()-28,left:0,width:cc.width(),height:28},collapse:{top:cc.height()}};
}
}
}
}
};
};
function _30(_31,_32){
var _33=$.data(_31,"layout").panels;
var _34=_35();
var p=_33[_32];
if(p.panel("options").onBeforeExpand.call(p)==false){
return;
}
var _36="expand"+_32.substring(0,1).toUpperCase()+_32.substring(1);
_33[_36].panel("close");
p.panel("panel").stop(true,true);
p.panel("expand",false).panel("open").panel("resize",_34.collapse);
p.panel("panel").animate(_34.expand,function(){
_2(_31);
});
function _35(){
var cc=$(_31);
if(_32=="east"&&_33.expandEast){
return {collapse:{left:cc.width()},expand:{left:cc.width()-_33["east"].panel("options").width}};
}else{
if(_32=="west"&&_33.expandWest){
return {collapse:{left:-_33["west"].panel("options").width},expand:{left:0}};
}else{
if(_32=="north"&&_33.expandNorth){
return {collapse:{top:-_33["north"].panel("options").height},expand:{top:0}};
}else{
if(_32=="south"&&_33.expandSouth){
return {collapse:{top:cc.height()},expand:{top:cc.height()-_33["south"].panel("options").height}};
}
}
}
}
};
};
function _37(_38){
var _39=$.data(_38,"layout").panels;
var cc=$(_38);
if(_39.east.length){
_39.east.panel("panel").bind("mouseover","east",_3a);
}
if(_39.west.length){
_39.west.panel("panel").bind("mouseover","west",_3a);
}
if(_39.north.length){
_39.north.panel("panel").bind("mouseover","north",_3a);
}
if(_39.south.length){
_39.south.panel("panel").bind("mouseover","south",_3a);
}
_39.center.panel("panel").bind("mouseover","center",_3a);
function _3a(e){
if(_1==true){
return;
}
if(e.data!="east"&&_b(_39.east)&&_b(_39.expandEast)){
_25(_38,"east");
}
if(e.data!="west"&&_b(_39.west)&&_b(_39.expandWest)){
_25(_38,"west");
}
if(e.data!="north"&&_b(_39.north)&&_b(_39.expandNorth)){
_25(_38,"north");
}
if(e.data!="south"&&_b(_39.south)&&_b(_39.expandSouth)){
_25(_38,"south");
}
return false;
};
};
function _b(pp){
if(!pp){
return false;
}
if(pp.length){
return pp.panel("panel").is(":visible");
}else{
return false;
}
};
function _3b(_3c){
var _3d=$.data(_3c,"layout").panels;
if(_3d.east.length&&_3d.east.panel("options").collapsed){
_25(_3c,"east",0);
}
if(_3d.west.length&&_3d.west.panel("options").collapsed){
_25(_3c,"west",0);
}
if(_3d.north.length&&_3d.north.panel("options").collapsed){
_25(_3c,"north",0);
}
if(_3d.south.length&&_3d.south.panel("options").collapsed){
_25(_3c,"south",0);
}
};
$.fn.layout=function(_3e,_3f){
if(typeof _3e=="string"){
return $.fn.layout.methods[_3e](this,_3f);
}
return this.each(function(){
var _40=$.data(this,"layout");
if(!_40){
var _41=$.extend({},{fit:$(this).attr("fit")=="true"});
$.data(this,"layout",{options:_41,panels:{center:$(),north:$(),south:$(),east:$(),west:$()}});
_c(this);
_37(this);
}
_2(this);
_3b(this);
});
};
$.fn.layout.methods={resize:function(jq){
return jq.each(function(){
_2(this);
});
},panel:function(jq,_42){
return $.data(jq[0],"layout").panels[_42];
},collapse:function(jq,_43){
return jq.each(function(){
_25(this,_43);
});
},expand:function(jq,_44){
return jq.each(function(){
_30(this,_44);
});
},add:function(jq,_45){
return jq.each(function(){
_11(this,_45);
_2(this);
if($(this).layout("panel",_45.region).panel("options").collapsed){
_25(this,_45.region,0);
}
});
},remove:function(jq,_46){
return jq.each(function(){
_20(this,_46);
_2(this);
});
}};
})(jQuery);

