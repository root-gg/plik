import{p as I}from"./chunk-JWPE2WC7.CBJ35mt7.js";import{s as _,g as F,p as D,o as E,a as z,b as P,_ as c,F as G,q as B,B as w,z as C,D as W,l as b,W as V,e as H}from"../app.Bsn4XIOt.js";import{p as j}from"./cynefin-VYW2F7L2.DVHOr5ic.js";import"./framework.DBSoX6Zw.js";import"./theme.DgxpkgsQ.js";var x={showLegend:!0,ticks:5,max:null,min:0,graticule:"circle"},y=32,A={axes:[],curves:[],options:x},m=structuredClone(A),U=W.radar,X=c(()=>w({...U,...C().radar}),"getConfig"),M=c(()=>m.axes,"getAxes"),q=c(()=>m.curves,"getCurves"),K=c(()=>m.options,"getOptions"),N=c(a=>{m.axes=a.map(t=>({name:t.name,label:t.label??t.name}))},"setAxes"),Y=c(a=>{m.curves=a.map(t=>({name:t.name,label:t.label??t.name,entries:Z(t.entries)}))},"setCurves"),Z=c(a=>{if(a[0].axis==null)return a.map(e=>e.value);const t=M();if(t.length===0)throw new Error("Axes must be populated before curves for reference entries");return t.map(e=>{const r=a.find(s=>{var n;return((n=s.axis)==null?void 0:n.$refText)===e.name});if(r===void 0)throw new Error("Missing entry for axis "+e.label);return r.value})},"computeCurveEntries"),J=c(a=>{var e,r,s,n,l;const t=a.reduce((o,i)=>(o[i.name]=i,o),{});m.options={showLegend:((e=t.showLegend)==null?void 0:e.value)??x.showLegend,ticks:((r=t.ticks)==null?void 0:r.value)??x.ticks,max:((s=t.max)==null?void 0:s.value)??x.max,min:((n=t.min)==null?void 0:n.value)??x.min,graticule:((l=t.graticule)==null?void 0:l.value)??x.graticule},m.options.ticks>y&&(b.warn(`Radar diagram ticks (${m.options.ticks}) exceeds maximum allowed (${y}). Using ${y} instead.`),m.options.ticks=y)},"setOptions"),Q=c(()=>{B(),m=structuredClone(A)},"clear"),$={getAxes:M,getCurves:q,getOptions:K,setAxes:N,setCurves:Y,setOptions:J,getConfig:X,clear:Q,setAccTitle:P,getAccTitle:z,setDiagramTitle:E,getDiagramTitle:D,getAccDescription:F,setAccDescription:_},tt=c(a=>{I(a,$);const{axes:t,curves:e,options:r}=a;$.setAxes(t),$.setCurves(e),$.setOptions(r)},"populate"),et={parse:c(async a=>{const t=await j("radar",a);b.debug(t),tt(t)},"parse")},at=c((a,t,e,r)=>{const s=r.db,n=s.getAxes(),l=s.getCurves(),o=s.getOptions(),i=s.getConfig(),d=s.getDiagramTitle(),p=G(t),u=rt(p,i),g=o.max??Math.max(...l.map(f=>Math.max(...f.entries))),h=o.min,v=Math.min(i.width,i.height)/2;st(u,n,v,o.ticks,o.graticule),nt(u,n,v,i),L(u,n,l,h,g,o.graticule,i),k(u,l,o.showLegend,i),u.append("text").attr("class","radarTitle").text(d).attr("x",0).attr("y",-i.height/2-i.marginTop)},"draw"),rt=c((a,t)=>{const e=t.width+t.marginLeft+t.marginRight,r=t.height+t.marginTop+t.marginBottom,s={x:t.marginLeft+t.width/2,y:t.marginTop+t.height/2};return H(a,r,e,t.useMaxWidth??!0),a.attr("viewBox",`0 0 ${e} ${r}`).attr("overflow","visible"),a.append("g").attr("transform",`translate(${s.x}, ${s.y})`)},"drawFrame"),st=c((a,t,e,r,s)=>{if(s==="circle")for(let n=0;n<r;n++){const l=e*(n+1)/r;a.append("circle").attr("r",l).attr("class","radarGraticule")}else if(s==="polygon"){const n=t.length;for(let l=0;l<r;l++){const o=e*(l+1)/r,i=t.map((d,p)=>{const u=2*p*Math.PI/n-Math.PI/2,g=o*Math.cos(u),h=o*Math.sin(u);return`${g},${h}`}).join(" ");a.append("polygon").attr("points",i).attr("class","radarGraticule")}}},"drawGraticule"),nt=c((a,t,e,r)=>{const s=t.length;for(let n=0;n<s;n++){const l=t[n].label,o=2*n*Math.PI/s-Math.PI/2,i=Math.cos(o),d=Math.sin(o);a.append("line").attr("x1",0).attr("y1",0).attr("x2",e*r.axisScaleFactor*i).attr("y2",e*r.axisScaleFactor*d).attr("class","radarAxisLine");const p=i>.01?"start":i<-.01?"end":"middle",u=d>.01?"hanging":d<-.01?"auto":"central",g=4;a.append("text").text(l).attr("x",e*r.axisLabelFactor*i+g*i).attr("y",e*r.axisLabelFactor*d+g*d).attr("text-anchor",p).attr("dominant-baseline",u).attr("class","radarAxisLabel")}},"drawAxes");function L(a,t,e,r,s,n,l){const o=t.length,i=Math.min(l.width,l.height)/2;e.forEach((d,p)=>{if(d.entries.length!==o)return;const u=d.entries.map((g,h)=>{const v=2*Math.PI*h/o-Math.PI/2,f=T(g,r,s,i),O=f*Math.cos(v),R=f*Math.sin(v);return{x:O,y:R}});n==="circle"?a.append("path").attr("d",S(u,l.curveTension)).attr("class",`radarCurve-${p}`):n==="polygon"&&a.append("polygon").attr("points",u.map(g=>`${g.x},${g.y}`).join(" ")).attr("class",`radarCurve-${p}`)})}c(L,"drawCurves");function T(a,t,e,r){const s=Math.min(Math.max(a,t),e);return r*(s-t)/(e-t)}c(T,"relativeRadius");function S(a,t){const e=a.length;let r=`M${a[0].x},${a[0].y}`;for(let s=0;s<e;s++){const n=a[(s-1+e)%e],l=a[s],o=a[(s+1)%e],i=a[(s+2)%e],d={x:l.x+(o.x-n.x)*t,y:l.y+(o.y-n.y)*t},p={x:o.x-(i.x-l.x)*t,y:o.y-(i.y-l.y)*t};r+=` C${d.x},${d.y} ${p.x},${p.y} ${o.x},${o.y}`}return`${r} Z`}c(S,"closedRoundCurve");function k(a,t,e,r){if(!e)return;const s=(r.width/2+r.marginRight)*3/4,n=-(r.height/2+r.marginTop)*3/4,l=20;t.forEach((o,i)=>{const d=a.append("g").attr("transform",`translate(${s}, ${n+i*l})`);d.append("rect").attr("width",12).attr("height",12).attr("class",`radarLegendBox-${i}`),d.append("text").attr("x",16).attr("y",0).attr("class","radarLegendText").text(o.label)})}c(k,"drawLegend");var ot={draw:at},it=c((a,t)=>{let e="";for(let r=0;r<a.THEME_COLOR_LIMIT;r++){const s=a[`cScale${r}`];e+=`
		.radarCurve-${r} {
			color: ${s};
			fill: ${s};
			fill-opacity: ${t.curveOpacity};
			stroke: ${s};
			stroke-width: ${t.curveStrokeWidth};
		}
		.radarLegendBox-${r} {
			fill: ${s};
			fill-opacity: ${t.curveOpacity};
			stroke: ${s};
		}
		`}return e},"genIndexStyles"),lt=c(a=>{const t=V(),e=C(),r=w(t,e.themeVariables),s=w(r.radar,a);return{themeVariables:r,radarOptions:s}},"buildRadarStyleOptions"),ct=c(({radar:a}={})=>{const{themeVariables:t,radarOptions:e}=lt(a);return`
	.radarTitle {
		font-size: ${t.fontSize};
		color: ${t.titleColor};
		dominant-baseline: hanging;
		text-anchor: middle;
	}
	.radarAxisLine {
		stroke: ${e.axisColor};
		stroke-width: ${e.axisStrokeWidth};
	}
	.radarAxisLabel {
		font-size: ${e.axisLabelFontSize}px;
		color: ${e.axisColor};
	}
	.radarGraticule {
		fill: ${e.graticuleColor};
		fill-opacity: ${e.graticuleOpacity};
		stroke: ${e.graticuleColor};
		stroke-width: ${e.graticuleStrokeWidth};
	}
	.radarLegendText {
		text-anchor: start;
		font-size: ${e.legendFontSize}px;
		dominant-baseline: hanging;
	}
	${it(t,e)}
	`},"styles"),xt={parser:et,db:$,renderer:ot,styles:ct};export{xt as diagram};
