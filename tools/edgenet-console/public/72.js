(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[72],{

/***/ "./resources/js/core/ResourceForm.js":
/*!*******************************************!*\
  !*** ./resources/js/core/ResourceForm.js ***!
  \*******************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var _form__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../form */ "./resources/js/form/index.js");



var ResourceForm = function ResourceForm(_ref) {
  var match = _ref.match;
  var ResourceForm = react__WEBPACK_IMPORTED_MODULE_0___default.a.lazy(function () {
    return __webpack_require__("./resources/js/views lazy recursive ^\\.\\/.*Form$")("./" + match.params.resource.charAt(0).toUpperCase() + match.params.resource.slice(1) + "Form")["catch"](function (err) {
      return {
        "default": function _default() {
          console.log(err);
          return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Not found");
        }
      };
    });
  });
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(react__WEBPACK_IMPORTED_MODULE_0__["Suspense"], {
    fallback: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Loading...")
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(_form__WEBPACK_IMPORTED_MODULE_1__["Form"], null, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ResourceForm, null)));
};

/* harmony default export */ __webpack_exports__["default"] = (ResourceForm);

/***/ }),

/***/ "./resources/js/form/index.js":
/*!************************************!*\
  !*** ./resources/js/form/index.js ***!
  \************************************/
/*! exports provided: Form, Related */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var _Form__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./Form */ "./resources/js/form/Form.js");
/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "Form", function() { return _Form__WEBPACK_IMPORTED_MODULE_0__["default"]; });

/* harmony import */ var _Related__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ./Related */ "./resources/js/form/Related.js");
/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "Related", function() { return _Related__WEBPACK_IMPORTED_MODULE_1__["default"]; });





/***/ }),

/***/ "./resources/js/views lazy recursive ^\\.\\/.*Form$":
/*!***************************************************************!*\
  !*** ./resources/js/views lazy ^\.\/.*Form$ namespace object ***!
  \***************************************************************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

var map = {
	"./GrammaireForm": [
		"./resources/js/views/GrammaireForm.js",
		18
	],
	"./NewsForm": [
		"./resources/js/views/NewsForm.js",
		1,
		4,
		2,
		13,
		17,
		84
	],
	"./NousForm": [
		"./resources/js/views/NousForm.js",
		21
	],
	"./ProjectsForm": [
		"./resources/js/views/ProjectsForm.js",
		23
	]
};
function webpackAsyncContext(req) {
	if(!__webpack_require__.o(map, req)) {
		return Promise.resolve().then(function() {
			var e = new Error("Cannot find module '" + req + "'");
			e.code = 'MODULE_NOT_FOUND';
			throw e;
		});
	}

	var ids = map[req], id = ids[0];
	return Promise.all(ids.slice(1).map(__webpack_require__.e)).then(function() {
		return __webpack_require__(id);
	});
}
webpackAsyncContext.keys = function webpackAsyncContextKeys() {
	return Object.keys(map);
};
webpackAsyncContext.id = "./resources/js/views lazy recursive ^\\.\\/.*Form$";
module.exports = webpackAsyncContext;

/***/ })

}]);