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
var _3=$(_2);
_3.addClass("tree");
return _3;
};
function _4(_5){
var _6=[];
_7(_6,$(_5));
function _7(aa,_8){
_8.children("li").each(function(){
var _9=$(this);
var _a={};
_a.text=_9.children("span").html();
if(!_a.text){
_a.text=_9.html();
}
_a.id=_9.attr("id");
_a.iconCls=_9.attr("iconCls")||_9.attr("icon");
_a.checked=_9.attr("checked")=="true";
_a.state=_9.attr("state")||"open";
var _b=_9.children("ul");
if(_b.length){
_a.children=[];
_7(_a.children,_b);
}
aa.push(_a);
});
};
return _6;
};
function _c(_d){
var _e=$.data(_d,"tree").options;
var _f=$.data(_d,"tree").tree;
$("div.tree-node",_f).unbind(".tree").bind("dblclick.tree",function(){
_c1(_d,this);
_e.onDblClick.call(_d,_9e(_d));
}).bind("click.tree",function(){
_c1(_d,this);
_e.onClick.call(_d,_9e(_d));
}).bind("mouseenter.tree",function(){
$(this).addClass("tree-node-hover");
return false;
}).bind("mouseleave.tree",function(){
$(this).removeClass("tree-node-hover");
return false;
}).bind("contextmenu.tree",function(e){
_e.onContextMenu.call(_d,e,_37(_d,this));
});
$("span.tree-hit",_f).unbind(".tree").bind("click.tree",function(){
var _10=$(this).parent();
_7b(_d,_10[0]);
return false;
}).bind("mouseenter.tree",function(){
if($(this).hasClass("tree-expanded")){
$(this).addClass("tree-expanded-hover");
}else{
$(this).addClass("tree-collapsed-hover");
}
}).bind("mouseleave.tree",function(){
if($(this).hasClass("tree-expanded")){
$(this).removeClass("tree-expanded-hover");
}else{
$(this).removeClass("tree-collapsed-hover");
}
}).bind("mousedown.tree",function(){
return false;
});
$("span.tree-checkbox",_f).unbind(".tree").bind("click.tree",function(){
var _11=$(this).parent();
_2e(_d,_11[0],!$(this).hasClass("tree-checkbox1"));
return false;
}).bind("mousedown.tree",function(){
return false;
});
};
function _12(_13){
var _14=$(_13).find("div.tree-node");
_14.draggable("disable");
_14.css("cursor","pointer");
};
function _15(_16){
var _17=$.data(_16,"tree").options;
var _18=$.data(_16,"tree").tree;
_18.find("div.tree-node").draggable({disabled:false,revert:true,cursor:"pointer",proxy:function(_19){
var p=$("<div class=\"tree-node-proxy tree-dnd-no\"></div>").appendTo("body");
p.html($(_19).find(".tree-title").html());
p.hide();
return p;
},deltaX:15,deltaY:15,onBeforeDrag:function(e){
if(e.which!=1){
return false;
}
$(this).next("ul").find("div.tree-node").droppable({accept:"no-accept"});
var _1a=$(this).find("span.tree-indent");
if(_1a.length){
e.data.startLeft+=_1a.length*_1a.width();
}
},onStartDrag:function(){
$(this).draggable("proxy").css({left:-10000,top:-10000});
},onDrag:function(e){
var x1=e.pageX,y1=e.pageY,x2=e.data.startX,y2=e.data.startY;
var d=Math.sqrt((x1-x2)*(x1-x2)+(y1-y2)*(y1-y2));
if(d>3){
$(this).draggable("proxy").show();
}
this.pageY=e.pageY;
},onStopDrag:function(){
$(this).next("ul").find("div.tree-node").droppable({accept:"div.tree-node"});
}}).droppable({accept:"div.tree-node",onDragOver:function(e,_1b){
var _1c=_1b.pageY;
var top=$(this).offset().top;
var _1d=top+$(this).outerHeight();
$(_1b).draggable("proxy").removeClass("tree-dnd-no").addClass("tree-dnd-yes");
$(this).removeClass("tree-node-append tree-node-top tree-node-bottom");
if(_1c>top+(_1d-top)/2){
if(_1d-_1c<5){
$(this).addClass("tree-node-bottom");
}else{
$(this).addClass("tree-node-append");
}
}else{
if(_1c-top<5){
$(this).addClass("tree-node-top");
}else{
$(this).addClass("tree-node-append");
}
}
},onDragLeave:function(e,_1e){
$(_1e).draggable("proxy").removeClass("tree-dnd-yes").addClass("tree-dnd-no");
$(this).removeClass("tree-node-append tree-node-top tree-node-bottom");
},onDrop:function(e,_1f){
var _20=this;
var _21,_22;
if($(this).hasClass("tree-node-append")){
_21=_23;
}else{
_21=_24;
_22=$(this).hasClass("tree-node-top")?"top":"bottom";
}
setTimeout(function(){
_21(_1f,_20,_22);
},0);
$(this).removeClass("tree-node-append tree-node-top tree-node-bottom");
}});
function _23(_25,_26){
if(_37(_16,_26).state=="closed"){
_6f(_16,_26,function(){
_27();
});
}else{
_27();
}
function _27(){
var _28=$(_16).tree("pop",_25);
$(_16).tree("append",{parent:_26,data:[_28]});
_17.onDrop.call(_16,_26,_28,"append");
};
};
function _24(_29,_2a,_2b){
var _2c={};
if(_2b=="top"){
_2c.before=_2a;
}else{
_2c.after=_2a;
}
var _2d=$(_16).tree("pop",_29);
_2c.data=_2d;
$(_16).tree("insert",_2c);
_17.onDrop.call(_16,_2a,_2d,_2b);
};
};
function _2e(_2f,_30,_31){
var _32=$.data(_2f,"tree").options;
if(!_32.checkbox){
return;
}
var _33=$(_30);
var ck=_33.find(".tree-checkbox");
ck.removeClass("tree-checkbox0 tree-checkbox1 tree-checkbox2");
if(_31){
ck.addClass("tree-checkbox1");
}else{
ck.addClass("tree-checkbox0");
}
if(_32.cascadeCheck){
_34(_33);
_35(_33);
}
var _36=_37(_2f,_30);
_32.onCheck.call(_2f,_36,_31);
function _35(_38){
var _39=_38.next().find(".tree-checkbox");
_39.removeClass("tree-checkbox0 tree-checkbox1 tree-checkbox2");
if(_38.find(".tree-checkbox").hasClass("tree-checkbox1")){
_39.addClass("tree-checkbox1");
}else{
_39.addClass("tree-checkbox0");
}
};
function _34(_3a){
var _3b=_86(_2f,_3a[0]);
if(_3b){
var ck=$(_3b.target).find(".tree-checkbox");
ck.removeClass("tree-checkbox0 tree-checkbox1 tree-checkbox2");
if(_3c(_3a)){
ck.addClass("tree-checkbox1");
}else{
if(_3d(_3a)){
ck.addClass("tree-checkbox0");
}else{
ck.addClass("tree-checkbox2");
}
}
_34($(_3b.target));
}
function _3c(n){
var ck=n.find(".tree-checkbox");
if(ck.hasClass("tree-checkbox0")||ck.hasClass("tree-checkbox2")){
return false;
}
var b=true;
n.parent().siblings().each(function(){
if(!$(this).children("div.tree-node").children(".tree-checkbox").hasClass("tree-checkbox1")){
b=false;
}
});
return b;
};
function _3d(n){
var ck=n.find(".tree-checkbox");
if(ck.hasClass("tree-checkbox1")||ck.hasClass("tree-checkbox2")){
return false;
}
var b=true;
n.parent().siblings().each(function(){
if(!$(this).children("div.tree-node").children(".tree-checkbox").hasClass("tree-checkbox0")){
b=false;
}
});
return b;
};
};
};
function _3e(_3f,_40){
var _41=$.data(_3f,"tree").options;
var _42=$(_40);
if(_43(_3f,_40)){
var ck=_42.find(".tree-checkbox");
if(ck.length){
if(ck.hasClass("tree-checkbox1")){
_2e(_3f,_40,true);
}else{
_2e(_3f,_40,false);
}
}else{
if(_41.onlyLeafCheck){
$("<span class=\"tree-checkbox tree-checkbox0\"></span>").insertBefore(_42.find(".tree-title"));
_c(_3f);
}
}
}else{
var ck=_42.find(".tree-checkbox");
if(_41.onlyLeafCheck){
ck.remove();
}else{
if(ck.hasClass("tree-checkbox1")){
_2e(_3f,_40,true);
}else{
if(ck.hasClass("tree-checkbox2")){
var _44=true;
var _45=true;
var _46=_47(_3f,_40);
for(var i=0;i<_46.length;i++){
if(_46[i].checked){
_45=false;
}else{
_44=false;
}
}
if(_44){
_2e(_3f,_40,true);
}
if(_45){
_2e(_3f,_40,false);
}
}
}
}
}
};
function _48(_49,ul,_4a,_4b){
var _4c=$.data(_49,"tree").options;
_4a=_4c.loadFilter.call(_49,_4a,$(ul).prev("div.tree-node")[0]);
if(!_4b){
$(ul).empty();
}
var _4d=[];
var _4e=$(ul).prev("div.tree-node").find("span.tree-indent, span.tree-hit").length;
_4f(ul,_4a,_4e);
_c(_49);
if(_4c.dnd){
_15(_49);
}else{
_12(_49);
}
for(var i=0;i<_4d.length;i++){
_2e(_49,_4d[i],true);
}
setTimeout(function(){
_57(_49,_49);
},0);
var _50=null;
if(_49!=ul){
var _51=$(ul).prev();
_50=_37(_49,_51[0]);
}
_4c.onLoadSuccess.call(_49,_50,_4a);
function _4f(ul,_52,_53){
for(var i=0;i<_52.length;i++){
var li=$("<li></li>").appendTo(ul);
var _54=_52[i];
if(_54.state!="open"&&_54.state!="closed"){
_54.state="open";
}
var _55=$("<div class=\"tree-node\"></div>").appendTo(li);
_55.attr("node-id",_54.id);
$.data(_55[0],"tree-node",{id:_54.id,text:_54.text,iconCls:_54.iconCls,attributes:_54.attributes});
$("<span class=\"tree-title\"></span>").html(_54.text).appendTo(_55);
if(_4c.checkbox){
if(_4c.onlyLeafCheck){
if(_54.state=="open"&&(!_54.children||!_54.children.length)){
if(_54.checked){
$("<span class=\"tree-checkbox tree-checkbox1\"></span>").prependTo(_55);
}else{
$("<span class=\"tree-checkbox tree-checkbox0\"></span>").prependTo(_55);
}
}
}else{
if(_54.checked){
$("<span class=\"tree-checkbox tree-checkbox1\"></span>").prependTo(_55);
_4d.push(_55[0]);
}else{
$("<span class=\"tree-checkbox tree-checkbox0\"></span>").prependTo(_55);
}
}
}
if(_54.children&&_54.children.length){
var _56=$("<ul></ul>").appendTo(li);
if(_54.state=="open"){
$("<span class=\"tree-icon tree-folder tree-folder-open\"></span>").addClass(_54.iconCls).prependTo(_55);
$("<span class=\"tree-hit tree-expanded\"></span>").prependTo(_55);
}else{
$("<span class=\"tree-icon tree-folder\"></span>").addClass(_54.iconCls).prependTo(_55);
$("<span class=\"tree-hit tree-collapsed\"></span>").prependTo(_55);
_56.css("display","none");
}
_4f(_56,_54.children,_53+1);
}else{
if(_54.state=="closed"){
$("<span class=\"tree-icon tree-folder\"></span>").addClass(_54.iconCls).prependTo(_55);
$("<span class=\"tree-hit tree-collapsed\"></span>").prependTo(_55);
}else{
$("<span class=\"tree-icon tree-file\"></span>").addClass(_54.iconCls).prependTo(_55);
$("<span class=\"tree-indent\"></span>").prependTo(_55);
}
}
for(var j=0;j<_53;j++){
$("<span class=\"tree-indent\"></span>").prependTo(_55);
}
}
};
};
function _57(_58,ul,_59){
var _5a=$.data(_58,"tree").options;
if(!_5a.lines){
return;
}
if(!_59){
_59=true;
$(_58).find("span.tree-indent").removeClass("tree-line tree-join tree-joinbottom");
$(_58).find("div.tree-node").removeClass("tree-node-last tree-root-first tree-root-one");
var _5b=$(_58).tree("getRoots");
if(_5b.length>1){
$(_5b[0].target).addClass("tree-root-first");
}else{
$(_5b[0].target).addClass("tree-root-one");
}
}
$(ul).children("li").each(function(){
var _5c=$(this).children("div.tree-node");
var ul=_5c.next("ul");
if(ul.length){
if($(this).next().length){
_5d(_5c);
}
_57(_58,ul,_59);
}else{
_5e(_5c);
}
});
var _5f=$(ul).children("li:last").children("div.tree-node").addClass("tree-node-last");
_5f.children("span.tree-join").removeClass("tree-join").addClass("tree-joinbottom");
function _5e(_60,_61){
var _62=_60.find("span.tree-icon");
_62.prev("span.tree-indent").addClass("tree-join");
};
function _5d(_63){
var _64=_63.find("span.tree-indent, span.tree-hit").length;
_63.next().find("div.tree-node").each(function(){
$(this).children("span:eq("+(_64-1)+")").addClass("tree-line");
});
};
};
function _65(_66,ul,_67,_68){
var _69=$.data(_66,"tree").options;
_67=_67||{};
var _6a=null;
if(_66!=ul){
var _6b=$(ul).prev();
_6a=_37(_66,_6b[0]);
}
if(_69.onBeforeLoad.call(_66,_6a,_67)==false){
return;
}
var _6c=$(ul).prev().children("span.tree-folder");
_6c.addClass("tree-loading");
var _6d=_69.loader.call(_66,_67,function(_6e){
_6c.removeClass("tree-loading");
_48(_66,ul,_6e);
if(_68){
_68();
}
},function(){
_6c.removeClass("tree-loading");
_69.onLoadError.apply(_66,arguments);
if(_68){
_68();
}
});
if(_6d==false){
_6c.removeClass("tree-loading");
}
};
function _6f(_70,_71,_72){
var _73=$.data(_70,"tree").options;
var hit=$(_71).children("span.tree-hit");
if(hit.length==0){
return;
}
if(hit.hasClass("tree-expanded")){
return;
}
var _74=_37(_70,_71);
if(_73.onBeforeExpand.call(_70,_74)==false){
return;
}
hit.removeClass("tree-collapsed tree-collapsed-hover").addClass("tree-expanded");
hit.next().addClass("tree-folder-open");
var ul=$(_71).next();
if(ul.length){
if(_73.animate){
ul.slideDown("normal",function(){
_73.onExpand.call(_70,_74);
if(_72){
_72();
}
});
}else{
ul.css("display","block");
_73.onExpand.call(_70,_74);
if(_72){
_72();
}
}
}else{
var _75=$("<ul style=\"display:none\"></ul>").insertAfter(_71);
_65(_70,_75[0],{id:_74.id},function(){
if(_75.is(":empty")){
_75.remove();
}
if(_73.animate){
_75.slideDown("normal",function(){
_73.onExpand.call(_70,_74);
if(_72){
_72();
}
});
}else{
_75.css("display","block");
_73.onExpand.call(_70,_74);
if(_72){
_72();
}
}
});
}
};
function _76(_77,_78){
var _79=$.data(_77,"tree").options;
var hit=$(_78).children("span.tree-hit");
if(hit.length==0){
return;
}
if(hit.hasClass("tree-collapsed")){
return;
}
var _7a=_37(_77,_78);
if(_79.onBeforeCollapse.call(_77,_7a)==false){
return;
}
hit.removeClass("tree-expanded tree-expanded-hover").addClass("tree-collapsed");
hit.next().removeClass("tree-folder-open");
var ul=$(_78).next();
if(_79.animate){
ul.slideUp("normal",function(){
_79.onCollapse.call(_77,_7a);
});
}else{
ul.css("display","none");
_79.onCollapse.call(_77,_7a);
}
};
function _7b(_7c,_7d){
var hit=$(_7d).children("span.tree-hit");
if(hit.length==0){
return;
}
if(hit.hasClass("tree-expanded")){
_76(_7c,_7d);
}else{
_6f(_7c,_7d);
}
};
function _7e(_7f,_80){
var _81=_47(_7f,_80);
if(_80){
_81.unshift(_37(_7f,_80));
}
for(var i=0;i<_81.length;i++){
_6f(_7f,_81[i].target);
}
};
function _82(_83,_84){
var _85=[];
var p=_86(_83,_84);
while(p){
_85.unshift(p);
p=_86(_83,p.target);
}
for(var i=0;i<_85.length;i++){
_6f(_83,_85[i].target);
}
};
function _87(_88,_89){
var _8a=_47(_88,_89);
if(_89){
_8a.unshift(_37(_88,_89));
}
for(var i=0;i<_8a.length;i++){
_76(_88,_8a[i].target);
}
};
function _8b(_8c){
var _8d=_8e(_8c);
if(_8d.length){
return _8d[0];
}else{
return null;
}
};
function _8e(_8f){
var _90=[];
$(_8f).children("li").each(function(){
var _91=$(this).children("div.tree-node");
_90.push(_37(_8f,_91[0]));
});
return _90;
};
function _47(_92,_93){
var _94=[];
if(_93){
_95($(_93));
}else{
var _96=_8e(_92);
for(var i=0;i<_96.length;i++){
_94.push(_96[i]);
_95($(_96[i].target));
}
}
function _95(_97){
_97.next().find("div.tree-node").each(function(){
_94.push(_37(_92,this));
});
};
return _94;
};
function _86(_98,_99){
var ul=$(_99).parent().parent();
if(ul[0]==_98){
return null;
}else{
return _37(_98,ul.prev()[0]);
}
};
function _9a(_9b){
var _9c=[];
$(_9b).find(".tree-checkbox1").each(function(){
var _9d=$(this).parent();
_9c.push(_37(_9b,_9d[0]));
});
return _9c;
};
function _9e(_9f){
var _a0=$(_9f).find("div.tree-node-selected");
if(_a0.length){
return _37(_9f,_a0[0]);
}else{
return null;
}
};
function _a1(_a2,_a3){
var _a4=$(_a3.parent);
var ul;
if(_a4.length==0){
ul=$(_a2);
}else{
ul=_a4.next();
if(ul.length==0){
ul=$("<ul></ul>").insertAfter(_a4);
}
}
if(_a3.data&&_a3.data.length){
var _a5=_a4.find("span.tree-icon");
if(_a5.hasClass("tree-file")){
_a5.removeClass("tree-file").addClass("tree-folder");
var hit=$("<span class=\"tree-hit tree-expanded\"></span>").insertBefore(_a5);
if(hit.prev().length){
hit.prev().remove();
}
}
}
_48(_a2,ul[0],_a3.data,true);
_3e(_a2,ul.prev());
};
function _a6(_a7,_a8){
var ref=_a8.before||_a8.after;
var _a9=_86(_a7,ref);
var li;
if(_a9){
_a1(_a7,{parent:_a9.target,data:[_a8.data]});
li=$(_a9.target).next().children("li:last");
}else{
_a1(_a7,{parent:null,data:[_a8.data]});
li=$(_a7).children("li:last");
}
if(_a8.before){
li.insertBefore($(ref).parent());
}else{
li.insertAfter($(ref).parent());
}
};
function _aa(_ab,_ac){
var _ad=_86(_ab,_ac);
var _ae=$(_ac);
var li=_ae.parent();
var ul=li.parent();
li.remove();
if(ul.children("li").length==0){
var _ae=ul.prev();
_ae.find(".tree-icon").removeClass("tree-folder").addClass("tree-file");
_ae.find(".tree-hit").remove();
$("<span class=\"tree-indent\"></span>").prependTo(_ae);
if(ul[0]!=_ab){
ul.remove();
}
}
if(_ad){
_3e(_ab,_ad.target);
}
_57(_ab,_ab);
};
function _af(_b0,_b1){
function _b2(aa,ul){
ul.children("li").each(function(){
var _b3=$(this).children("div.tree-node");
var _b4=_37(_b0,_b3[0]);
var sub=$(this).children("ul");
if(sub.length){
_b4.children=[];
_af(_b4.children,sub);
}
aa.push(_b4);
});
};
if(_b1){
var _b5=_37(_b0,_b1);
_b5.children=[];
_b2(_b5.children,$(_b1).next());
return _b5;
}else{
return null;
}
};
function _b6(_b7,_b8){
var _b9=$(_b8.target);
var _ba=$.data(_b8.target,"tree-node");
if(_ba.iconCls){
_b9.find(".tree-icon").removeClass(_ba.iconCls);
}
$.extend(_ba,_b8);
$.data(_b8.target,"tree-node",_ba);
_b9.attr("node-id",_ba.id);
_b9.find(".tree-title").html(_ba.text);
if(_ba.iconCls){
_b9.find(".tree-icon").addClass(_ba.iconCls);
}
var ck=_b9.find(".tree-checkbox");
ck.removeClass("tree-checkbox0 tree-checkbox1 tree-checkbox2");
if(_ba.checked){
_2e(_b7,_b8.target,true);
}else{
_2e(_b7,_b8.target,false);
}
};
function _37(_bb,_bc){
var _bd=$.extend({},$.data(_bc,"tree-node"),{target:_bc,checked:$(_bc).find(".tree-checkbox").hasClass("tree-checkbox1")});
if(!_43(_bb,_bc)){
_bd.state=$(_bc).find(".tree-hit").hasClass("tree-expanded")?"open":"closed";
}
return _bd;
};
function _be(_bf,id){
var _c0=$(_bf).find("div.tree-node[node-id="+id+"]");
if(_c0.length){
return _37(_bf,_c0[0]);
}else{
return null;
}
};
function _c1(_c2,_c3){
var _c4=$.data(_c2,"tree").options;
var _c5=_37(_c2,_c3);
if(_c4.onBeforeSelect.call(_c2,_c5)==false){
return;
}
$("div.tree-node-selected",_c2).removeClass("tree-node-selected");
$(_c3).addClass("tree-node-selected");
_c4.onSelect.call(_c2,_c5);
};
function _43(_c6,_c7){
var _c8=$(_c7);
var hit=_c8.children("span.tree-hit");
return hit.length==0;
};
function _c9(_ca,_cb){
var _cc=$.data(_ca,"tree").options;
var _cd=_37(_ca,_cb);
if(_cc.onBeforeEdit.call(_ca,_cd)==false){
return;
}
$(_cb).css("position","relative");
var nt=$(_cb).find(".tree-title");
var _ce=nt.outerWidth();
nt.empty();
var _cf=$("<input class=\"tree-editor\">").appendTo(nt);
_cf.val(_cd.text).focus();
_cf.width(_ce+20);
_cf.height(document.compatMode=="CSS1Compat"?(18-(_cf.outerHeight()-_cf.height())):18);
_cf.bind("click",function(e){
return false;
}).bind("mousedown",function(e){
e.stopPropagation();
}).bind("mousemove",function(e){
e.stopPropagation();
}).bind("keydown",function(e){
if(e.keyCode==13){
_d0(_ca,_cb);
return false;
}else{
if(e.keyCode==27){
_d6(_ca,_cb);
return false;
}
}
}).bind("blur",function(e){
e.stopPropagation();
_d0(_ca,_cb);
});
};
function _d0(_d1,_d2){
var _d3=$.data(_d1,"tree").options;
$(_d2).css("position","");
var _d4=$(_d2).find("input.tree-editor");
var val=_d4.val();
_d4.remove();
var _d5=_37(_d1,_d2);
_d5.text=val;
_b6(_d1,_d5);
_d3.onAfterEdit.call(_d1,_d5);
};
function _d6(_d7,_d8){
var _d9=$.data(_d7,"tree").options;
$(_d8).css("position","");
$(_d8).find("input.tree-editor").remove();
var _da=_37(_d7,_d8);
_b6(_d7,_da);
_d9.onCancelEdit.call(_d7,_da);
};
$.fn.tree=function(_db,_dc){
if(typeof _db=="string"){
return $.fn.tree.methods[_db](this,_dc);
}
var _db=_db||{};
return this.each(function(){
var _dd=$.data(this,"tree");
var _de;
if(_dd){
_de=$.extend(_dd.options,_db);
_dd.options=_de;
}else{
_de=$.extend({},$.fn.tree.defaults,$.fn.tree.parseOptions(this),_db);
$.data(this,"tree",{options:_de,tree:_1(this)});
var _df=_4(this);
if(_df.length&&!_de.data){
_de.data=_df;
}
}
if(_de.lines){
$(this).addClass("tree-lines");
}
if(_de.data){
_48(this,this,_de.data);
}else{
if(_de.dnd){
_15(this);
}else{
_12(this);
}
}
_65(this,this);
});
};
$.fn.tree.methods={options:function(jq){
return $.data(jq[0],"tree").options;
},loadData:function(jq,_e0){
return jq.each(function(){
_48(this,this,_e0);
});
},getNode:function(jq,_e1){
return _37(jq[0],_e1);
},getData:function(jq,_e2){
return _af(jq[0],_e2);
},reload:function(jq,_e3){
return jq.each(function(){
if(_e3){
var _e4=$(_e3);
var hit=_e4.children("span.tree-hit");
hit.removeClass("tree-expanded tree-expanded-hover").addClass("tree-collapsed");
_e4.next().remove();
_6f(this,_e3);
}else{
$(this).empty();
_65(this,this);
}
});
},getRoot:function(jq){
return _8b(jq[0]);
},getRoots:function(jq){
return _8e(jq[0]);
},getParent:function(jq,_e5){
return _86(jq[0],_e5);
},getChildren:function(jq,_e6){
return _47(jq[0],_e6);
},getChecked:function(jq){
return _9a(jq[0]);
},getSelected:function(jq){
return _9e(jq[0]);
},isLeaf:function(jq,_e7){
return _43(jq[0],_e7);
},find:function(jq,id){
return _be(jq[0],id);
},select:function(jq,_e8){
return jq.each(function(){
_c1(this,_e8);
});
},check:function(jq,_e9){
return jq.each(function(){
_2e(this,_e9,true);
});
},uncheck:function(jq,_ea){
return jq.each(function(){
_2e(this,_ea,false);
});
},collapse:function(jq,_eb){
return jq.each(function(){
_76(this,_eb);
});
},expand:function(jq,_ec){
return jq.each(function(){
_6f(this,_ec);
});
},collapseAll:function(jq,_ed){
return jq.each(function(){
_87(this,_ed);
});
},expandAll:function(jq,_ee){
return jq.each(function(){
_7e(this,_ee);
});
},expandTo:function(jq,_ef){
return jq.each(function(){
_82(this,_ef);
});
},toggle:function(jq,_f0){
return jq.each(function(){
_7b(this,_f0);
});
},append:function(jq,_f1){
return jq.each(function(){
_a1(this,_f1);
});
},insert:function(jq,_f2){
return jq.each(function(){
_a6(this,_f2);
});
},remove:function(jq,_f3){
return jq.each(function(){
_aa(this,_f3);
});
},pop:function(jq,_f4){
var _f5=jq.tree("getData",_f4);
jq.tree("remove",_f4);
return _f5;
},update:function(jq,_f6){
return jq.each(function(){
_b6(this,_f6);
});
},enableDnd:function(jq){
return jq.each(function(){
_15(this);
});
},disableDnd:function(jq){
return jq.each(function(){
_12(this);
});
},beginEdit:function(jq,_f7){
return jq.each(function(){
_c9(this,_f7);
});
},endEdit:function(jq,_f8){
return jq.each(function(){
_d0(this,_f8);
});
},cancelEdit:function(jq,_f9){
return jq.each(function(){
_d6(this,_f9);
});
}};
$.fn.tree.parseOptions=function(_fa){
var t=$(_fa);
return {url:t.attr("url"),method:(t.attr("method")?t.attr("method"):undefined),checkbox:(t.attr("checkbox")?t.attr("checkbox")=="true":undefined),cascadeCheck:(t.attr("cascadeCheck")?t.attr("cascadeCheck")=="true":undefined),onlyLeafCheck:(t.attr("onlyLeafCheck")?t.attr("onlyLeafCheck")=="true":undefined),animate:(t.attr("animate")?t.attr("animate")=="true":undefined),lines:(t.attr("lines")?t.attr("lines")=="true":undefined),dnd:(t.attr("dnd")?t.attr("dnd")=="true":undefined)};
};
$.fn.tree.defaults={url:null,method:"post",animate:false,checkbox:false,cascadeCheck:true,onlyLeafCheck:false,lines:false,dnd:false,data:null,loader:function(_fb,_fc,_fd){
var _fe=$(this).tree("options");
if(!_fe.url){
return false;
}
$.ajax({type:_fe.method,url:_fe.url,data:_fb,dataType:"json",success:function(_ff){
_fc(_ff);
},error:function(){
_fd.apply(this,arguments);
}});
},loadFilter:function(data,_100){
return data;
},onBeforeLoad:function(node,_101){
},onLoadSuccess:function(node,data){
},onLoadError:function(){
},onClick:function(node){
},onDblClick:function(node){
},onBeforeExpand:function(node){
},onExpand:function(node){
},onBeforeCollapse:function(node){
},onCollapse:function(node){
},onCheck:function(node,_102){
},onBeforeSelect:function(node){
},onSelect:function(node){
},onContextMenu:function(e,node){
},onDrop:function(_103,_104,_105){
},onBeforeEdit:function(node){
},onAfterEdit:function(node){
},onCancelEdit:function(node){
}};
})(jQuery);

