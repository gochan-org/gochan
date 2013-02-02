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
var _3=$.data(_2,"accordion").options;
var _4=$.data(_2,"accordion").panels;
var cc=$(_2);
if(_3.fit==true){
var p=cc.parent();
p.addClass("panel-noscroll");
_3.width=p.width();
_3.height=p.height();
}
if(_3.width>0){
cc._outerWidth(_3.width);
}
var _5="auto";
if(_3.height>0){
cc._outerHeight(_3.height);
var _6=_4.length?_4[0].panel("header").css("height","").outerHeight():"auto";
var _5=cc.height()-(_4.length-1)*_6;
}
for(var i=0;i<_4.length;i++){
var _7=_4[i];
var _8=_7.panel("header");
_8._outerHeight(_6);
_7.panel("resize",{width:cc.width(),height:_5});
}
};
function _9(_a){
var _b=$.data(_a,"accordion").panels;
for(var i=0;i<_b.length;i++){
var _c=_b[i];
if(_c.panel("options").collapsed==false){
return _c;
}
}
return null;
};
function _d(_e,_f,_10){
var _11=$.data(_e,"accordion").panels;
for(var i=0;i<_11.length;i++){
var _12=_11[i];
if(_12.panel("options").title==_f){
if(_10){
_11.splice(i,1);
}
return _12;
}
}
return null;
};
function _13(_14){
var cc=$(_14);
cc.addClass("accordion");
if(cc.attr("border")=="false"){
cc.addClass("accordion-noborder");
}else{
cc.removeClass("accordion-noborder");
}
var _15=cc.children("div[selected]");
cc.children("div").not(_15).attr("collapsed","true");
if(_15.length==0){
cc.children("div:first").attr("collapsed","false");
}
var _16=[];
cc.children("div").each(function(){
var pp=$(this);
_16.push(pp);
_19(_14,pp,{});
});
cc.bind("_resize",function(e,_17){
var _18=$.data(_14,"accordion").options;
if(_18.fit==true||_17){
_1(_14);
}
return false;
});
return {accordion:cc,panels:_16};
};
function _19(_1a,pp,_1b){
pp.panel($.extend({},_1b,{collapsible:false,minimizable:false,maximizable:false,closable:false,doSize:false,tools:[{iconCls:"accordion-collapse",handler:function(){
var _1c=$.data(_1a,"accordion").options.animate;
if(pp.panel("options").collapsed){
_28(_1a);
pp.panel("expand",_1c);
}else{
_28(_1a);
pp.panel("collapse",_1c);
}
return false;
}}],onBeforeExpand:function(){
var _1d=_9(_1a);
if(_1d){
var _1e=$(_1d).panel("header");
_1e.removeClass("accordion-header-selected");
_1e.find(".accordion-collapse").triggerHandler("click");
}
var _1e=pp.panel("header");
_1e.addClass("accordion-header-selected");
_1e.find(".accordion-collapse").removeClass("accordion-expand");
},onExpand:function(){
var _1f=$.data(_1a,"accordion").options;
_1f.onSelect.call(_1a,pp.panel("options").title);
},onBeforeCollapse:function(){
var _20=pp.panel("header");
_20.removeClass("accordion-header-selected");
_20.find(".accordion-collapse").addClass("accordion-expand");
}}));
pp.panel("body").addClass("accordion-body");
pp.panel("header").addClass("accordion-header").click(function(){
$(this).find(".accordion-collapse").triggerHandler("click");
return false;
});
};
function _21(_22,_23){
var _24=$.data(_22,"accordion").options;
var _25=$.data(_22,"accordion").panels;
var _26=_9(_22);
if(_26&&_26.panel("options").title==_23){
return;
}
var _27=_d(_22,_23);
if(_27){
_27.panel("header").triggerHandler("click");
}else{
if(_26){
_26.panel("header").addClass("accordion-header-selected");
_24.onSelect.call(_22,_26.panel("options").title);
}
}
};
function _28(_29){
var _2a=$.data(_29,"accordion").panels;
for(var i=0;i<_2a.length;i++){
_2a[i].stop(true,true);
}
};
function add(_2b,_2c){
var _2d=$.data(_2b,"accordion").options;
var _2e=$.data(_2b,"accordion").panels;
_28(_2b);
_2c.collapsed=_2c.selected==undefined?true:_2c.selected;
var pp=$("<div></div>").appendTo(_2b);
_2e.push(pp);
_19(_2b,pp,_2c);
_1(_2b);
_2d.onAdd.call(_2b,_2c.title);
_21(_2b,_2c.title);
};
function _2f(_30,_31){
var _32=$.data(_30,"accordion").options;
var _33=$.data(_30,"accordion").panels;
_28(_30);
if(_32.onBeforeRemove.call(_30,_31)==false){
return;
}
var _34=_d(_30,_31,true);
if(_34){
_34.panel("destroy");
if(_33.length){
_1(_30);
var _35=_9(_30);
if(!_35){
_21(_30,_33[0].panel("options").title);
}
}
}
_32.onRemove.call(_30,_31);
};
$.fn.accordion=function(_36,_37){
if(typeof _36=="string"){
return $.fn.accordion.methods[_36](this,_37);
}
_36=_36||{};
return this.each(function(){
var _38=$.data(this,"accordion");
var _39;
if(_38){
_39=$.extend(_38.options,_36);
_38.opts=_39;
}else{
_39=$.extend({},$.fn.accordion.defaults,$.fn.accordion.parseOptions(this),_36);
var r=_13(this);
$.data(this,"accordion",{options:_39,accordion:r.accordion,panels:r.panels});
}
_1(this);
_21(this);
});
};
$.fn.accordion.methods={options:function(jq){
return $.data(jq[0],"accordion").options;
},panels:function(jq){
return $.data(jq[0],"accordion").panels;
},resize:function(jq){
return jq.each(function(){
_1(this);
});
},getSelected:function(jq){
return _9(jq[0]);
},getPanel:function(jq,_3a){
return _d(jq[0],_3a);
},select:function(jq,_3b){
return jq.each(function(){
_21(this,_3b);
});
},add:function(jq,_3c){
return jq.each(function(){
add(this,_3c);
});
},remove:function(jq,_3d){
return jq.each(function(){
_2f(this,_3d);
});
}};
$.fn.accordion.parseOptions=function(_3e){
var t=$(_3e);
return {width:(parseInt(_3e.style.width)||undefined),height:(parseInt(_3e.style.height)||undefined),fit:(t.attr("fit")?t.attr("fit")=="true":undefined),border:(t.attr("border")?t.attr("border")=="true":undefined),animate:(t.attr("animate")?t.attr("animate")=="true":undefined)};
};
$.fn.accordion.defaults={width:"auto",height:"auto",fit:false,border:true,animate:true,onSelect:function(_3f){
},onAdd:function(_40){
},onBeforeRemove:function(_41){
},onRemove:function(_42){
}};
})(jQuery);

