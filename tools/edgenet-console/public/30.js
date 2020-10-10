(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[30],{

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

/***/ "./resources/js/form/old/FormFieldImage.js":
/*!*************************************************!*\
  !*** ./resources/js/form/old/FormFieldImage.js ***!
  \*************************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var grommet_icons__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! grommet-icons */ "./node_modules/grommet-icons/es6/index.js");
!(function webpackMissingModule() { var e = new Error("Cannot find module '../DataSource'"); e.code = 'MODULE_NOT_FOUND'; throw e; }());
/* harmony import */ var filesize__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! filesize */ "./node_modules/filesize/lib/filesize.min.js");
/* harmony import */ var filesize__WEBPACK_IMPORTED_MODULE_4___default = /*#__PURE__*/__webpack_require__.n(filesize__WEBPACK_IMPORTED_MODULE_4__);
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







var ImageField = function ImageField(_ref) {
  var file = _ref.file,
      preview = _ref.preview,
      onClick = _ref.onClick;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    direction: "row",
    gap: "small"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Anchor"], {
    onClick: onClick
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    width: "small",
    height: "small"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Image"], {
    src: preview,
    fit: "cover",
    title: file.name
  }))), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], null, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Text"], {
    size: "small"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Anchor"], {
    onClick: onClick
  }, file.name)), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Text"], {
    size: "small"
  }, filesize__WEBPACK_IMPORTED_MODULE_4___default()(file.size))));
};

var FormFieldImage = /*#__PURE__*/function (_React$Component) {
  _inherits(FormFieldImage, _React$Component);

  var _super = _createSuper(FormFieldImage);

  function FormFieldImage(props) {
    var _this;

    _classCallCheck(this, FormFieldImage);

    _this = _super.call(this, props);
    _this.state = {
      file: null,
      preview: null
    };
    _this.selectImage = _this.selectImage.bind(_assertThisInitialized(_this));
    return _this;
  }
  /*
      Called after the system dialog to select an image
      TODO: properties are name, size, type
  */


  _createClass(FormFieldImage, [{
    key: "selectImage",
    value: function selectImage(event) {
      var _this2 = this;

      var attachFile = this.context.attachFile;
      var file = null;

      if (event.target.files && event.target.files[0]) {
        file = event.target.files[0];
        var reader = new FileReader();

        reader.onloadend = function () {
          return _this2.setState({
            preview: reader.result,
            file: file
          }, function () {
            return attachFile(file);
          });
        };

        reader.readAsDataURL(file);
      }
    }
  }, {
    key: "render",
    value: function render() {
      var _this3 = this;

      var name = this.props.name;
      var _this$state = this.state,
          file = _this$state.file,
          preview = _this$state.preview;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        pad: {
          vertical: 'small'
        }
      }, file ? /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ImageField, {
        file: file,
        preview: preview,
        onClick: function onClick() {
          return _this3.inputElement.click();
        }
      }) : /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Button"], {
        icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_2__["Add"], null),
        label: "S\xE9lectionner une image",
        plain: true,
        onClick: function onClick() {
          return _this3.inputElement.click();
        }
      }), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("input", {
        type: "file",
        onChange: this.selectImage,
        ref: function ref(_ref2) {
          _this3.inputElement = _ref2;
        },
        id: name,
        name: name,
        hidden: true
      }));
    }
  }]);

  return FormFieldImage;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

FormFieldImage.contextType = !(function webpackMissingModule() { var e = new Error("Cannot find module '../DataSource'"); e.code = 'MODULE_NOT_FOUND'; throw e; }());
/* harmony default export */ __webpack_exports__["default"] = (FormFieldImage);

/***/ })

}]);