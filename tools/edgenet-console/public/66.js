(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[66],{

/***/ "./node_modules/filesize/lib/filesize.min.js":
/*!***************************************************!*\
  !*** ./node_modules/filesize/lib/filesize.min.js ***!
  \***************************************************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

"use strict";
/* WEBPACK VAR INJECTION */(function(global) {/*
 2020 Jason Mulligan <jason.mulligan@avoidwork.com>
 @version 6.1.0
*/
!function(e){var x=/^(b|B)$/,M={iec:{bits:["b","Kib","Mib","Gib","Tib","Pib","Eib","Zib","Yib"],bytes:["B","KiB","MiB","GiB","TiB","PiB","EiB","ZiB","YiB"]},jedec:{bits:["b","Kb","Mb","Gb","Tb","Pb","Eb","Zb","Yb"],bytes:["B","KB","MB","GB","TB","PB","EB","ZB","YB"]}},w={iec:["","kibi","mebi","gibi","tebi","pebi","exbi","zebi","yobi"],jedec:["","kilo","mega","giga","tera","peta","exa","zetta","yotta"]};function t(e){var i,t,o,n,b,r,a,l,s,d,u,c,f,p,B,y=1<arguments.length&&void 0!==arguments[1]?arguments[1]:{},g=[],v=0,m=void 0,h=void 0;if(isNaN(e))throw new TypeError("Invalid number");return t=!0===y.bits,u=!0===y.unix,i=y.base||2,d=void 0!==y.round?y.round:u?1:2,r=void 0!==y.locale?y.locale:"",a=y.localeOptions||{},c=void 0!==y.separator?y.separator:"",f=void 0!==y.spacer?y.spacer:u?"":" ",B=y.symbols||{},p=2===i&&y.standard||"jedec",s=y.output||"string",n=!0===y.fullform,b=y.fullforms instanceof Array?y.fullforms:[],m=void 0!==y.exponent?y.exponent:-1,o=2<i?1e3:1024,(l=(h=Number(e))<0)&&(h=-h),(-1===m||isNaN(m))&&(m=Math.floor(Math.log(h)/Math.log(o)))<0&&(m=0),8<m&&(m=8),"exponent"===s?m:(0===h?(g[0]=0,g[1]=u?"":M[p][t?"bits":"bytes"][m]):(v=h/(2===i?Math.pow(2,10*m):Math.pow(1e3,m)),t&&o<=(v*=8)&&m<8&&(v/=o,m++),g[0]=Number(v.toFixed(0<m?d:0)),g[0]===o&&m<8&&void 0===y.exponent&&(g[0]=1,m++),g[1]=10===i&&1===m?t?"kb":"kB":M[p][t?"bits":"bytes"][m],u&&(g[1]="jedec"===p?g[1].charAt(0):0<m?g[1].replace(/B$/,""):g[1],x.test(g[1])&&(g[0]=Math.floor(g[0]),g[1]=""))),l&&(g[0]=-g[0]),g[1]=B[g[1]]||g[1],!0===r?g[0]=g[0].toLocaleString():0<r.length?g[0]=g[0].toLocaleString(r,a):0<c.length&&(g[0]=g[0].toString().replace(".",c)),"array"===s?g:(n&&(g[1]=b[m]?b[m]:w[p][m]+(t?"bit":"byte")+(1===g[0]?"":"s")),"object"===s?{value:g[0],symbol:g[1],exponent:m}:g.join(f)))}t.partial=function(i){return function(e){return t(e,i)}}, true?module.exports=t:undefined}("undefined"!=typeof window?window:global);
//# sourceMappingURL=filesize.min.js.map
/* WEBPACK VAR INJECTION */}.call(this, __webpack_require__(/*! ./../../webpack/buildin/global.js */ "./node_modules/webpack/buildin/global.js")))

/***/ }),

/***/ "./node_modules/moment/locale sync recursive ^\\.\\/.*$":
/*!**************************************************!*\
  !*** ./node_modules/moment/locale sync ^\.\/.*$ ***!
  \**************************************************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

var map = {
	"./af": "./node_modules/moment/locale/af.js",
	"./af.js": "./node_modules/moment/locale/af.js",
	"./ar": "./node_modules/moment/locale/ar.js",
	"./ar-dz": "./node_modules/moment/locale/ar-dz.js",
	"./ar-dz.js": "./node_modules/moment/locale/ar-dz.js",
	"./ar-kw": "./node_modules/moment/locale/ar-kw.js",
	"./ar-kw.js": "./node_modules/moment/locale/ar-kw.js",
	"./ar-ly": "./node_modules/moment/locale/ar-ly.js",
	"./ar-ly.js": "./node_modules/moment/locale/ar-ly.js",
	"./ar-ma": "./node_modules/moment/locale/ar-ma.js",
	"./ar-ma.js": "./node_modules/moment/locale/ar-ma.js",
	"./ar-sa": "./node_modules/moment/locale/ar-sa.js",
	"./ar-sa.js": "./node_modules/moment/locale/ar-sa.js",
	"./ar-tn": "./node_modules/moment/locale/ar-tn.js",
	"./ar-tn.js": "./node_modules/moment/locale/ar-tn.js",
	"./ar.js": "./node_modules/moment/locale/ar.js",
	"./az": "./node_modules/moment/locale/az.js",
	"./az.js": "./node_modules/moment/locale/az.js",
	"./be": "./node_modules/moment/locale/be.js",
	"./be.js": "./node_modules/moment/locale/be.js",
	"./bg": "./node_modules/moment/locale/bg.js",
	"./bg.js": "./node_modules/moment/locale/bg.js",
	"./bm": "./node_modules/moment/locale/bm.js",
	"./bm.js": "./node_modules/moment/locale/bm.js",
	"./bn": "./node_modules/moment/locale/bn.js",
	"./bn.js": "./node_modules/moment/locale/bn.js",
	"./bo": "./node_modules/moment/locale/bo.js",
	"./bo.js": "./node_modules/moment/locale/bo.js",
	"./br": "./node_modules/moment/locale/br.js",
	"./br.js": "./node_modules/moment/locale/br.js",
	"./bs": "./node_modules/moment/locale/bs.js",
	"./bs.js": "./node_modules/moment/locale/bs.js",
	"./ca": "./node_modules/moment/locale/ca.js",
	"./ca.js": "./node_modules/moment/locale/ca.js",
	"./cs": "./node_modules/moment/locale/cs.js",
	"./cs.js": "./node_modules/moment/locale/cs.js",
	"./cv": "./node_modules/moment/locale/cv.js",
	"./cv.js": "./node_modules/moment/locale/cv.js",
	"./cy": "./node_modules/moment/locale/cy.js",
	"./cy.js": "./node_modules/moment/locale/cy.js",
	"./da": "./node_modules/moment/locale/da.js",
	"./da.js": "./node_modules/moment/locale/da.js",
	"./de": "./node_modules/moment/locale/de.js",
	"./de-at": "./node_modules/moment/locale/de-at.js",
	"./de-at.js": "./node_modules/moment/locale/de-at.js",
	"./de-ch": "./node_modules/moment/locale/de-ch.js",
	"./de-ch.js": "./node_modules/moment/locale/de-ch.js",
	"./de.js": "./node_modules/moment/locale/de.js",
	"./dv": "./node_modules/moment/locale/dv.js",
	"./dv.js": "./node_modules/moment/locale/dv.js",
	"./el": "./node_modules/moment/locale/el.js",
	"./el.js": "./node_modules/moment/locale/el.js",
	"./en-au": "./node_modules/moment/locale/en-au.js",
	"./en-au.js": "./node_modules/moment/locale/en-au.js",
	"./en-ca": "./node_modules/moment/locale/en-ca.js",
	"./en-ca.js": "./node_modules/moment/locale/en-ca.js",
	"./en-gb": "./node_modules/moment/locale/en-gb.js",
	"./en-gb.js": "./node_modules/moment/locale/en-gb.js",
	"./en-ie": "./node_modules/moment/locale/en-ie.js",
	"./en-ie.js": "./node_modules/moment/locale/en-ie.js",
	"./en-il": "./node_modules/moment/locale/en-il.js",
	"./en-il.js": "./node_modules/moment/locale/en-il.js",
	"./en-in": "./node_modules/moment/locale/en-in.js",
	"./en-in.js": "./node_modules/moment/locale/en-in.js",
	"./en-nz": "./node_modules/moment/locale/en-nz.js",
	"./en-nz.js": "./node_modules/moment/locale/en-nz.js",
	"./en-sg": "./node_modules/moment/locale/en-sg.js",
	"./en-sg.js": "./node_modules/moment/locale/en-sg.js",
	"./eo": "./node_modules/moment/locale/eo.js",
	"./eo.js": "./node_modules/moment/locale/eo.js",
	"./es": "./node_modules/moment/locale/es.js",
	"./es-do": "./node_modules/moment/locale/es-do.js",
	"./es-do.js": "./node_modules/moment/locale/es-do.js",
	"./es-us": "./node_modules/moment/locale/es-us.js",
	"./es-us.js": "./node_modules/moment/locale/es-us.js",
	"./es.js": "./node_modules/moment/locale/es.js",
	"./et": "./node_modules/moment/locale/et.js",
	"./et.js": "./node_modules/moment/locale/et.js",
	"./eu": "./node_modules/moment/locale/eu.js",
	"./eu.js": "./node_modules/moment/locale/eu.js",
	"./fa": "./node_modules/moment/locale/fa.js",
	"./fa.js": "./node_modules/moment/locale/fa.js",
	"./fi": "./node_modules/moment/locale/fi.js",
	"./fi.js": "./node_modules/moment/locale/fi.js",
	"./fil": "./node_modules/moment/locale/fil.js",
	"./fil.js": "./node_modules/moment/locale/fil.js",
	"./fo": "./node_modules/moment/locale/fo.js",
	"./fo.js": "./node_modules/moment/locale/fo.js",
	"./fr": "./node_modules/moment/locale/fr.js",
	"./fr-ca": "./node_modules/moment/locale/fr-ca.js",
	"./fr-ca.js": "./node_modules/moment/locale/fr-ca.js",
	"./fr-ch": "./node_modules/moment/locale/fr-ch.js",
	"./fr-ch.js": "./node_modules/moment/locale/fr-ch.js",
	"./fr.js": "./node_modules/moment/locale/fr.js",
	"./fy": "./node_modules/moment/locale/fy.js",
	"./fy.js": "./node_modules/moment/locale/fy.js",
	"./ga": "./node_modules/moment/locale/ga.js",
	"./ga.js": "./node_modules/moment/locale/ga.js",
	"./gd": "./node_modules/moment/locale/gd.js",
	"./gd.js": "./node_modules/moment/locale/gd.js",
	"./gl": "./node_modules/moment/locale/gl.js",
	"./gl.js": "./node_modules/moment/locale/gl.js",
	"./gom-deva": "./node_modules/moment/locale/gom-deva.js",
	"./gom-deva.js": "./node_modules/moment/locale/gom-deva.js",
	"./gom-latn": "./node_modules/moment/locale/gom-latn.js",
	"./gom-latn.js": "./node_modules/moment/locale/gom-latn.js",
	"./gu": "./node_modules/moment/locale/gu.js",
	"./gu.js": "./node_modules/moment/locale/gu.js",
	"./he": "./node_modules/moment/locale/he.js",
	"./he.js": "./node_modules/moment/locale/he.js",
	"./hi": "./node_modules/moment/locale/hi.js",
	"./hi.js": "./node_modules/moment/locale/hi.js",
	"./hr": "./node_modules/moment/locale/hr.js",
	"./hr.js": "./node_modules/moment/locale/hr.js",
	"./hu": "./node_modules/moment/locale/hu.js",
	"./hu.js": "./node_modules/moment/locale/hu.js",
	"./hy-am": "./node_modules/moment/locale/hy-am.js",
	"./hy-am.js": "./node_modules/moment/locale/hy-am.js",
	"./id": "./node_modules/moment/locale/id.js",
	"./id.js": "./node_modules/moment/locale/id.js",
	"./is": "./node_modules/moment/locale/is.js",
	"./is.js": "./node_modules/moment/locale/is.js",
	"./it": "./node_modules/moment/locale/it.js",
	"./it-ch": "./node_modules/moment/locale/it-ch.js",
	"./it-ch.js": "./node_modules/moment/locale/it-ch.js",
	"./it.js": "./node_modules/moment/locale/it.js",
	"./ja": "./node_modules/moment/locale/ja.js",
	"./ja.js": "./node_modules/moment/locale/ja.js",
	"./jv": "./node_modules/moment/locale/jv.js",
	"./jv.js": "./node_modules/moment/locale/jv.js",
	"./ka": "./node_modules/moment/locale/ka.js",
	"./ka.js": "./node_modules/moment/locale/ka.js",
	"./kk": "./node_modules/moment/locale/kk.js",
	"./kk.js": "./node_modules/moment/locale/kk.js",
	"./km": "./node_modules/moment/locale/km.js",
	"./km.js": "./node_modules/moment/locale/km.js",
	"./kn": "./node_modules/moment/locale/kn.js",
	"./kn.js": "./node_modules/moment/locale/kn.js",
	"./ko": "./node_modules/moment/locale/ko.js",
	"./ko.js": "./node_modules/moment/locale/ko.js",
	"./ku": "./node_modules/moment/locale/ku.js",
	"./ku.js": "./node_modules/moment/locale/ku.js",
	"./ky": "./node_modules/moment/locale/ky.js",
	"./ky.js": "./node_modules/moment/locale/ky.js",
	"./lb": "./node_modules/moment/locale/lb.js",
	"./lb.js": "./node_modules/moment/locale/lb.js",
	"./lo": "./node_modules/moment/locale/lo.js",
	"./lo.js": "./node_modules/moment/locale/lo.js",
	"./lt": "./node_modules/moment/locale/lt.js",
	"./lt.js": "./node_modules/moment/locale/lt.js",
	"./lv": "./node_modules/moment/locale/lv.js",
	"./lv.js": "./node_modules/moment/locale/lv.js",
	"./me": "./node_modules/moment/locale/me.js",
	"./me.js": "./node_modules/moment/locale/me.js",
	"./mi": "./node_modules/moment/locale/mi.js",
	"./mi.js": "./node_modules/moment/locale/mi.js",
	"./mk": "./node_modules/moment/locale/mk.js",
	"./mk.js": "./node_modules/moment/locale/mk.js",
	"./ml": "./node_modules/moment/locale/ml.js",
	"./ml.js": "./node_modules/moment/locale/ml.js",
	"./mn": "./node_modules/moment/locale/mn.js",
	"./mn.js": "./node_modules/moment/locale/mn.js",
	"./mr": "./node_modules/moment/locale/mr.js",
	"./mr.js": "./node_modules/moment/locale/mr.js",
	"./ms": "./node_modules/moment/locale/ms.js",
	"./ms-my": "./node_modules/moment/locale/ms-my.js",
	"./ms-my.js": "./node_modules/moment/locale/ms-my.js",
	"./ms.js": "./node_modules/moment/locale/ms.js",
	"./mt": "./node_modules/moment/locale/mt.js",
	"./mt.js": "./node_modules/moment/locale/mt.js",
	"./my": "./node_modules/moment/locale/my.js",
	"./my.js": "./node_modules/moment/locale/my.js",
	"./nb": "./node_modules/moment/locale/nb.js",
	"./nb.js": "./node_modules/moment/locale/nb.js",
	"./ne": "./node_modules/moment/locale/ne.js",
	"./ne.js": "./node_modules/moment/locale/ne.js",
	"./nl": "./node_modules/moment/locale/nl.js",
	"./nl-be": "./node_modules/moment/locale/nl-be.js",
	"./nl-be.js": "./node_modules/moment/locale/nl-be.js",
	"./nl.js": "./node_modules/moment/locale/nl.js",
	"./nn": "./node_modules/moment/locale/nn.js",
	"./nn.js": "./node_modules/moment/locale/nn.js",
	"./oc-lnc": "./node_modules/moment/locale/oc-lnc.js",
	"./oc-lnc.js": "./node_modules/moment/locale/oc-lnc.js",
	"./pa-in": "./node_modules/moment/locale/pa-in.js",
	"./pa-in.js": "./node_modules/moment/locale/pa-in.js",
	"./pl": "./node_modules/moment/locale/pl.js",
	"./pl.js": "./node_modules/moment/locale/pl.js",
	"./pt": "./node_modules/moment/locale/pt.js",
	"./pt-br": "./node_modules/moment/locale/pt-br.js",
	"./pt-br.js": "./node_modules/moment/locale/pt-br.js",
	"./pt.js": "./node_modules/moment/locale/pt.js",
	"./ro": "./node_modules/moment/locale/ro.js",
	"./ro.js": "./node_modules/moment/locale/ro.js",
	"./ru": "./node_modules/moment/locale/ru.js",
	"./ru.js": "./node_modules/moment/locale/ru.js",
	"./sd": "./node_modules/moment/locale/sd.js",
	"./sd.js": "./node_modules/moment/locale/sd.js",
	"./se": "./node_modules/moment/locale/se.js",
	"./se.js": "./node_modules/moment/locale/se.js",
	"./si": "./node_modules/moment/locale/si.js",
	"./si.js": "./node_modules/moment/locale/si.js",
	"./sk": "./node_modules/moment/locale/sk.js",
	"./sk.js": "./node_modules/moment/locale/sk.js",
	"./sl": "./node_modules/moment/locale/sl.js",
	"./sl.js": "./node_modules/moment/locale/sl.js",
	"./sq": "./node_modules/moment/locale/sq.js",
	"./sq.js": "./node_modules/moment/locale/sq.js",
	"./sr": "./node_modules/moment/locale/sr.js",
	"./sr-cyrl": "./node_modules/moment/locale/sr-cyrl.js",
	"./sr-cyrl.js": "./node_modules/moment/locale/sr-cyrl.js",
	"./sr.js": "./node_modules/moment/locale/sr.js",
	"./ss": "./node_modules/moment/locale/ss.js",
	"./ss.js": "./node_modules/moment/locale/ss.js",
	"./sv": "./node_modules/moment/locale/sv.js",
	"./sv.js": "./node_modules/moment/locale/sv.js",
	"./sw": "./node_modules/moment/locale/sw.js",
	"./sw.js": "./node_modules/moment/locale/sw.js",
	"./ta": "./node_modules/moment/locale/ta.js",
	"./ta.js": "./node_modules/moment/locale/ta.js",
	"./te": "./node_modules/moment/locale/te.js",
	"./te.js": "./node_modules/moment/locale/te.js",
	"./tet": "./node_modules/moment/locale/tet.js",
	"./tet.js": "./node_modules/moment/locale/tet.js",
	"./tg": "./node_modules/moment/locale/tg.js",
	"./tg.js": "./node_modules/moment/locale/tg.js",
	"./th": "./node_modules/moment/locale/th.js",
	"./th.js": "./node_modules/moment/locale/th.js",
	"./tl-ph": "./node_modules/moment/locale/tl-ph.js",
	"./tl-ph.js": "./node_modules/moment/locale/tl-ph.js",
	"./tlh": "./node_modules/moment/locale/tlh.js",
	"./tlh.js": "./node_modules/moment/locale/tlh.js",
	"./tr": "./node_modules/moment/locale/tr.js",
	"./tr.js": "./node_modules/moment/locale/tr.js",
	"./tzl": "./node_modules/moment/locale/tzl.js",
	"./tzl.js": "./node_modules/moment/locale/tzl.js",
	"./tzm": "./node_modules/moment/locale/tzm.js",
	"./tzm-latn": "./node_modules/moment/locale/tzm-latn.js",
	"./tzm-latn.js": "./node_modules/moment/locale/tzm-latn.js",
	"./tzm.js": "./node_modules/moment/locale/tzm.js",
	"./ug-cn": "./node_modules/moment/locale/ug-cn.js",
	"./ug-cn.js": "./node_modules/moment/locale/ug-cn.js",
	"./uk": "./node_modules/moment/locale/uk.js",
	"./uk.js": "./node_modules/moment/locale/uk.js",
	"./ur": "./node_modules/moment/locale/ur.js",
	"./ur.js": "./node_modules/moment/locale/ur.js",
	"./uz": "./node_modules/moment/locale/uz.js",
	"./uz-latn": "./node_modules/moment/locale/uz-latn.js",
	"./uz-latn.js": "./node_modules/moment/locale/uz-latn.js",
	"./uz.js": "./node_modules/moment/locale/uz.js",
	"./vi": "./node_modules/moment/locale/vi.js",
	"./vi.js": "./node_modules/moment/locale/vi.js",
	"./x-pseudo": "./node_modules/moment/locale/x-pseudo.js",
	"./x-pseudo.js": "./node_modules/moment/locale/x-pseudo.js",
	"./yo": "./node_modules/moment/locale/yo.js",
	"./yo.js": "./node_modules/moment/locale/yo.js",
	"./zh-cn": "./node_modules/moment/locale/zh-cn.js",
	"./zh-cn.js": "./node_modules/moment/locale/zh-cn.js",
	"./zh-hk": "./node_modules/moment/locale/zh-hk.js",
	"./zh-hk.js": "./node_modules/moment/locale/zh-hk.js",
	"./zh-mo": "./node_modules/moment/locale/zh-mo.js",
	"./zh-mo.js": "./node_modules/moment/locale/zh-mo.js",
	"./zh-tw": "./node_modules/moment/locale/zh-tw.js",
	"./zh-tw.js": "./node_modules/moment/locale/zh-tw.js"
};


function webpackContext(req) {
	var id = webpackContextResolve(req);
	return __webpack_require__(id);
}
function webpackContextResolve(req) {
	if(!__webpack_require__.o(map, req)) {
		var e = new Error("Cannot find module '" + req + "'");
		e.code = 'MODULE_NOT_FOUND';
		throw e;
	}
	return map[req];
}
webpackContext.keys = function webpackContextKeys() {
	return Object.keys(map);
};
webpackContext.resolve = webpackContextResolve;
module.exports = webpackContext;
webpackContext.id = "./node_modules/moment/locale sync recursive ^\\.\\/.*$";

/***/ }),

/***/ "./resources/js/data/index.js":
/*!************************************!*\
  !*** ./resources/js/data/index.js ***!
  \************************************/
/*! exports provided: Data, DataConsumer, DataContext */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var _Data__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./Data */ "./resources/js/data/Data.js");
/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "Data", function() { return _Data__WEBPACK_IMPORTED_MODULE_0__["Data"]; });

/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "DataConsumer", function() { return _Data__WEBPACK_IMPORTED_MODULE_0__["DataConsumer"]; });

/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "DataContext", function() { return _Data__WEBPACK_IMPORTED_MODULE_0__["DataContext"]; });




/***/ }),

/***/ "./resources/js/data/views/List.js":
/*!*****************************************!*\
  !*** ./resources/js/data/views/List.js ***!
  \*****************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var ___WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../. */ "./resources/js/data/index.js");
/* harmony import */ var _order__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../order */ "./resources/js/data/order/index.js");
function _typeof(obj) { "@babel/helpers - typeof"; if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof = function _typeof(obj) { return typeof obj; }; } else { _typeof = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof(obj); }

function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

function _defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ("value" in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } }

function _createClass(Constructor, protoProps, staticProps) { if (protoProps) _defineProperties(Constructor.prototype, protoProps); if (staticProps) _defineProperties(Constructor, staticProps); return Constructor; }

function _inherits(subClass, superClass) { if (typeof superClass !== "function" && superClass !== null) { throw new TypeError("Super expression must either be null or a function"); } subClass.prototype = Object.create(superClass && superClass.prototype, { constructor: { value: subClass, writable: true, configurable: true } }); if (superClass) _setPrototypeOf(subClass, superClass); }

function _setPrototypeOf(o, p) { _setPrototypeOf = Object.setPrototypeOf || function _setPrototypeOf(o, p) { o.__proto__ = p; return o; }; return _setPrototypeOf(o, p); }

function _createSuper(Derived) { return function () { var Super = _getPrototypeOf(Derived), result; if (_isNativeReflectConstruct()) { var NewTarget = _getPrototypeOf(this).constructor; result = Reflect.construct(Super, arguments, NewTarget); } else { result = Super.apply(this, arguments); } return _possibleConstructorReturn(this, result); }; }

function _possibleConstructorReturn(self, call) { if (call && (_typeof(call) === "object" || typeof call === "function")) { return call; } return _assertThisInitialized(self); }

function _assertThisInitialized(self) { if (self === void 0) { throw new ReferenceError("this hasn't been initialised - super() hasn't been called"); } return self; }

function _isNativeReflectConstruct() { if (typeof Reflect === "undefined" || !Reflect.construct) return false; if (Reflect.construct.sham) return false; if (typeof Proxy === "function") return true; try { Date.prototype.toString.call(Reflect.construct(Date, [], function () {})); return true; } catch (e) { return false; } }

function _getPrototypeOf(o) { _getPrototypeOf = Object.setPrototypeOf ? Object.getPrototypeOf : function _getPrototypeOf(o) { return o.__proto__ || Object.getPrototypeOf(o); }; return _getPrototypeOf(o); }






var Loading = function Loading() {
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    flex: "grow",
    justify: "center",
    align: "center"
  }, "...");
};

var ListRow = /*#__PURE__*/function (_React$Component) {
  _inherits(ListRow, _React$Component);

  var _super = _createSuper(ListRow);

  function ListRow(props) {
    var _this;

    _classCallCheck(this, ListRow);

    _this = _super.call(this, props);
    _this.state = {
      isMouseOver: false
    };
    return _this;
  }

  _createClass(ListRow, [{
    key: "render",
    value: function render() {
      var _this2 = this;

      var children = this.props.children;
      var _this$props = this.props,
          item = _this$props.item,
          isActive = _this$props.isActive,
          orderable = _this$props.orderable,
          _onClick = _this$props.onClick;
      var isMouseOver = this.state.isMouseOver;
      var background = isActive ? 'light-4' : isMouseOver ? 'light-2' : 'light-1';
      children = react__WEBPACK_IMPORTED_MODULE_0___default.a.cloneElement(children, {
        item: item,
        isActive: isActive,
        isMouseOver: isMouseOver
      }, null);

      if (orderable) {
        children = /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(_order__WEBPACK_IMPORTED_MODULE_3__["OrderableItem"], {
          isMouseOver: isMouseOver,
          item: item
        }, children);
      }

      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        onMouseEnter: function onMouseEnter() {
          return _this2.setState({
            isMouseOver: true
          });
        },
        onMouseLeave: function onMouseLeave() {
          return _this2.setState({
            isMouseOver: false
          });
        },
        onClick: function onClick() {
          return _onClick(item);
        },
        background: background,
        border: {
          side: 'bottom',
          color: 'light-4'
        },
        flex: false
      }, children);
    }
  }]);

  return ListRow;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

var List = function List(_ref) {
  var children = _ref.children,
      onClick = _ref.onClick,
      _ref$show = _ref.show,
      show = _ref$show === void 0 ? false : _ref$show;
  //
  // let ListComponent = null;
  // if (component) {
  //     ListComponent = component;
  // } else if (children) {
  //     ListComponent = React.cloneElement(children, null);
  // } else {
  //     return null;
  // }
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(___WEBPACK_IMPORTED_MODULE_2__["DataConsumer"], null, function (_ref2) {
    var identifier = _ref2.identifier,
        items = _ref2.items,
        per_page = _ref2.per_page,
        itemsLoading = _ref2.itemsLoading,
        getItem = _ref2.getItem,
        currentId = _ref2.currentId,
        getItems = _ref2.getItems,
        selectable = _ref2.selectable,
        orderable = _ref2.orderable;

    if (itemsLoading) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Loading, null);
    }

    if (items.length === 0) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        pad: "small",
        align: "center"
      }, "No items found");
    }

    if (currentId) {
      if (!isNaN(currentId)) {
        // check if it is a number
        currentId = parseInt(currentId);
      }
    }

    var currentIdx = null;

    if (show && currentId) {
      currentIdx = items.findIndex(function (item) {
        return item[identifier] === currentId;
      });
    }

    return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
      overflow: "auto"
    }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["InfiniteScroll"], {
      items: items,
      onMore: getItems,
      step: per_page // show={currentIdx}
      // renderMarker={marker => itemsLoading && <Box pad="medium" background="accent-1">{marker}</Box>}

    }, function (item, j) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ListRow, {
        key: 'items-' + j,
        item: item,
        onClick: onClick === undefined ? getItem : onClick,
        selectable: selectable,
        orderable: orderable,
        isActive: item[identifier] === currentId
      }, children);
    }));
  });
};

/* harmony default export */ __webpack_exports__["default"] = (List);

/***/ }),

/***/ "./resources/js/data/views/index.js":
/*!******************************************!*\
  !*** ./resources/js/data/views/index.js ***!
  \******************************************/
/*! exports provided: List */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var _List__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./List */ "./resources/js/data/views/List.js");
/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "List", function() { return _List__WEBPACK_IMPORTED_MODULE_0__["default"]; });




/***/ }),

/***/ "./resources/js/form/tabs/Images.js":
/*!******************************************!*\
  !*** ./resources/js/form/tabs/Images.js ***!
  \******************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! axios */ "./node_modules/axios/index.js");
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_1___default = /*#__PURE__*/__webpack_require__.n(axios__WEBPACK_IMPORTED_MODULE_1__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var grommet_icons__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! grommet-icons */ "./node_modules/grommet-icons/es6/index.js");
/* harmony import */ var moment__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! moment */ "./node_modules/moment/moment.js");
/* harmony import */ var moment__WEBPACK_IMPORTED_MODULE_4___default = /*#__PURE__*/__webpack_require__.n(moment__WEBPACK_IMPORTED_MODULE_4__);
/* harmony import */ var filesize__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! filesize */ "./node_modules/filesize/lib/filesize.min.js");
/* harmony import */ var filesize__WEBPACK_IMPORTED_MODULE_5___default = /*#__PURE__*/__webpack_require__.n(filesize__WEBPACK_IMPORTED_MODULE_5__);
/* harmony import */ var _data_views__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../../data/views */ "./resources/js/data/views/index.js");
/* harmony import */ var _data__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../../data */ "./resources/js/data/index.js");
function _typeof(obj) { "@babel/helpers - typeof"; if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof = function _typeof(obj) { return typeof obj; }; } else { _typeof = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof(obj); }

function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

function _defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ("value" in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } }

function _createClass(Constructor, protoProps, staticProps) { if (protoProps) _defineProperties(Constructor.prototype, protoProps); if (staticProps) _defineProperties(Constructor, staticProps); return Constructor; }

function _inherits(subClass, superClass) { if (typeof superClass !== "function" && superClass !== null) { throw new TypeError("Super expression must either be null or a function"); } subClass.prototype = Object.create(superClass && superClass.prototype, { constructor: { value: subClass, writable: true, configurable: true } }); if (superClass) _setPrototypeOf(subClass, superClass); }

function _setPrototypeOf(o, p) { _setPrototypeOf = Object.setPrototypeOf || function _setPrototypeOf(o, p) { o.__proto__ = p; return o; }; return _setPrototypeOf(o, p); }

function _createSuper(Derived) { return function () { var Super = _getPrototypeOf(Derived), result; if (_isNativeReflectConstruct()) { var NewTarget = _getPrototypeOf(this).constructor; result = Reflect.construct(Super, arguments, NewTarget); } else { result = Super.apply(this, arguments); } return _possibleConstructorReturn(this, result); }; }

function _possibleConstructorReturn(self, call) { if (call && (_typeof(call) === "object" || typeof call === "function")) { return call; } return _assertThisInitialized(self); }

function _assertThisInitialized(self) { if (self === void 0) { throw new ReferenceError("this hasn't been initialised - super() hasn't been called"); } return self; }

function _isNativeReflectConstruct() { if (typeof Reflect === "undefined" || !Reflect.construct) return false; if (Reflect.construct.sham) return false; if (typeof Proxy === "function") return true; try { Date.prototype.toString.call(Reflect.construct(Date, [], function () {})); return true; } catch (e) { return false; } }

function _getPrototypeOf(o) { _getPrototypeOf = Object.setPrototypeOf ? Object.getPrototypeOf : function _getPrototypeOf(o) { return o.__proto__ || Object.getPrototypeOf(o); }; return _getPrototypeOf(o); }










var ImageList = function ImageList(_ref) {
  var item = _ref.item;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
    direction: "row",
    gap: "small",
    pad: "small",
    flex: "grow"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
    height: "xsmall",
    width: "xsmall"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Image"], {
    src: item.thumb,
    fit: "cover",
    title: item.title
  })), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
    flex: "grow"
  }, item.title && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Text"], {
    size: "small"
  }, item.title), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], null), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Text"], {
    size: "small"
  }, item.name), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Text"], {
    size: "small"
  }, filesize__WEBPACK_IMPORTED_MODULE_5___default()(item.size))), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
    align: "end"
  }));
};

var ImagePreview = /*#__PURE__*/function (_React$Component) {
  _inherits(ImagePreview, _React$Component);

  var _super = _createSuper(ImagePreview);

  function ImagePreview(props) {
    var _this;

    _classCallCheck(this, ImagePreview);

    _this = _super.call(this, props);
    _this.state = {
      preview: null
    };
    _this.reader = new FileReader();
    return _this;
  }

  _createClass(ImagePreview, [{
    key: "componentDidMount",
    value: function componentDidMount() {
      var _this2 = this;

      var file = this.props.file;

      this.reader.onloadend = function () {
        return _this2.setState({
          preview: _this2.reader.result,
          file: file
        });
      };

      this.reader.readAsDataURL(file);
      this.onClose = this.onClose.bind(this);
    }
  }, {
    key: "onClose",
    value: function onClose() {
      var onClose = this.props.onClose;
      this.setState({
        preview: null
      }, onClose);
    }
  }, {
    key: "render",
    value: function render() {
      var _this$props = this.props,
          file = _this$props.file,
          onSave = _this$props.onSave,
          onDelete = _this$props.onDelete,
          progress = _this$props.progress;
      var preview = this.state.preview;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Layer"], {
        position: "center",
        modal: true,
        animate: false,
        onEsc: this.onClose,
        onClickOutside: this.onClose
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Stack"], {
        fill: true
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        height: "large"
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Image"], {
        src: preview,
        fit: "contain",
        fill: "horizontal",
        title: file.name
      })), progress > 0 && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        align: "center",
        justify: "center",
        fill: true,
        background: {
          'color': 'dark-1',
          'opacity': 'strong'
        }
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Text"], null, "Uploading, please wait..."), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Meter"], {
        values: [{
          'value': progress
        }]
      }))), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        pad: "small",
        grow: "2"
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Text"], {
        size: "small"
      }, "Filename: ", file.name), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Text"], {
        size: "small"
      }, "Size: ", filesize__WEBPACK_IMPORTED_MODULE_5___default()(file.size)), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        justify: "end",
        direction: "row",
        gap: "small"
      }, onDelete && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
        plain: true,
        label: "Effacer",
        icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_3__["Trash"], {
          color: "status-critical"
        }),
        onClick: onDelete
      }), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
        primary: true,
        disabled: progress > 0,
        label: "Sauvgarder",
        icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_3__["Save"], null),
        onClick: onSave
      }))));
    }
  }]);

  return ImagePreview;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

var UploadImage = /*#__PURE__*/function (_React$Component2) {
  _inherits(UploadImage, _React$Component2);

  var _super2 = _createSuper(UploadImage);

  function UploadImage(props) {
    var _this3;

    _classCallCheck(this, UploadImage);

    _this3 = _super2.call(this, props);
    _this3.state = {
      file: null,
      progress: 0,
      message: ''
    };
    _this3.selectImage = _this3.selectImage.bind(_assertThisInitialized(_this3));
    _this3.upload = _this3.upload.bind(_assertThisInitialized(_this3));
    return _this3;
  }
  /*
      Called after the system dialog to select an image
      TODO: properties are name, size, type
  */


  _createClass(UploadImage, [{
    key: "selectImage",
    value: function selectImage(event) {
      if (event.target.files && event.target.files[0]) {
        this.setState({
          file: event.target.files[0]
        });
      }
    }
  }, {
    key: "upload",
    value: function upload() {
      var _this4 = this;

      var file = this.state.file;
      var name = this.props.name;
      var _this$context = this.context,
          url = _this$context.url,
          pushItem = _this$context.pushItem;
      var data = new FormData();
      data.append(name, file);
      var config = {
        onUploadProgress: function onUploadProgress(ev) {
          return _this4.setState({
            progress: Math.round(ev.loaded * 100 / ev.total)
          });
        }
      };
      axios__WEBPACK_IMPORTED_MODULE_1___default.a.post(url, data, config).then(function (_ref2) {
        var data = _ref2.data;
        pushItem(data.data);

        _this4.setState({
          progress: 0,
          file: null
        });
      })["catch"](function (err) {
        return _this4.setState({
          progress: 0,
          message: err.message
        });
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this5 = this;

      var name = this.props.name;
      var _this$state = this.state,
          file = _this$state.file,
          progress = _this$state.progress,
          message = _this$state.message;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        pad: {
          vertical: 'small'
        }
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], null, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
        icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_3__["Add"], null),
        label: "S\xE9lectionner une image",
        plain: true,
        onClick: function onClick() {
          return _this5.inputElement.click();
        }
      }), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("input", {
        type: "file",
        onChange: this.selectImage,
        ref: function ref(_ref3) {
          _this5.inputElement = _ref3;
        },
        id: name,
        name: name,
        hidden: true
      })), message, file && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ImagePreview, {
        file: file,
        progress: progress,
        onSave: this.upload,
        onClose: function onClose() {
          return _this5.setState({
            file: null
          });
        }
      }));
    }
  }]);

  return UploadImage;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

UploadImage.contextType = _data__WEBPACK_IMPORTED_MODULE_7__["DataContext"];

var ModifyImage = /*#__PURE__*/function (_React$Component3) {
  _inherits(ModifyImage, _React$Component3);

  var _super3 = _createSuper(ModifyImage);

  function ModifyImage(props) {
    var _this6;

    _classCallCheck(this, ModifyImage);

    _this6 = _super3.call(this, props);
    _this6.state = {
      image: null,
      preview: null,
      confirmDelete: false
    };
    _this6.handleClose = _this6.handleClose.bind(_assertThisInitialized(_this6));
    _this6.handleDelete = _this6.handleDelete.bind(_assertThisInitialized(_this6));
    return _this6;
  }

  _createClass(ModifyImage, [{
    key: "componentDidMount",
    value: function componentDidMount() {
      var _this7 = this;

      var image = this.props.image;
      axios__WEBPACK_IMPORTED_MODULE_1___default.a.get(image.url, {
        responseType: 'blob'
      }).then(function (_ref4) {
        var data = _ref4.data;
        return _this7.setState({
          image: image,
          preview: new Blob([data])
        });
      })["catch"](function (err) {
        return console.log(err);
      });
    }
  }, {
    key: "handleClose",
    value: function handleClose() {
      var onClose = this.props.onClose;
      this.setState({
        preview: null
      }, onClose);
    }
  }, {
    key: "handleDelete",
    value: function handleDelete() {
      var image = this.state.image;
      var _this$context2 = this.context,
          url = _this$context2.url,
          pullItem = _this$context2.pullItem;
      if (!image && !image.id) return;
      axios__WEBPACK_IMPORTED_MODULE_1___default.a["delete"](url + "/" + image.id).then(function () {
        return pullItem(image);
      }).then(this.handleClose);
    }
  }, {
    key: "render",
    value: function render() {
      var preview = this.state.preview;
      return preview && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ImagePreview, {
        file: preview,
        onDelete: this.handleDelete,
        onClose: this.handleClose
      });
    }
  }]);

  return ModifyImage;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

ModifyImage.contextType = _data__WEBPACK_IMPORTED_MODULE_7__["DataContext"];

var Images = /*#__PURE__*/function (_React$Component4) {
  _inherits(Images, _React$Component4);

  var _super4 = _createSuper(Images);

  function Images(props) {
    var _this8;

    _classCallCheck(this, Images);

    _this8 = _super4.call(this, props);
    _this8.state = {
      image: null
    };
    _this8.setImage = _this8.setImage.bind(_assertThisInitialized(_this8));
    _this8.unsetImage = _this8.unsetImage.bind(_assertThisInitialized(_this8));
    return _this8;
  }

  _createClass(Images, [{
    key: "setImage",
    value: function setImage(image) {
      this.setState({
        image: image
      });
    }
  }, {
    key: "unsetImage",
    value: function unsetImage() {
      this.setState({
        image: null
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this$props2 = this.props,
          resource = _this$props2.resource,
          id = _this$props2.id;
      var image = this.state.image;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(_data__WEBPACK_IMPORTED_MODULE_7__["Data"], {
        orderable: true,
        url: "/api/" + resource + "/" + id + "/images"
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(_data_views__WEBPACK_IMPORTED_MODULE_6__["List"], {
        onClick: this.setImage
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ImageList, null)), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(UploadImage, {
        name: "image"
      }), image && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ModifyImage, {
        image: image,
        onClose: this.unsetImage
      }));
    }
  }]);

  return Images;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

Images.title = 'Images';
/* harmony default export */ __webpack_exports__["default"] = (Images);

/***/ })

}]);