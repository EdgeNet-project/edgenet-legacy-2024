(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[73],{

/***/ "./resources/js/core/ResourceView.js":
/*!*******************************************!*\
  !*** ./resources/js/core/ResourceView.js ***!
  \*******************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var _view__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../view */ "./resources/js/view/index.js");



var ResourceView = function ResourceView(_ref) {
  var match = _ref.match;
  var ResourceView = react__WEBPACK_IMPORTED_MODULE_0___default.a.lazy(function () {
    return __webpack_require__("./resources/js/views lazy recursive ^\\.\\/.*View$")("./" + match.params.resource.charAt(0).toUpperCase() + match.params.resource.slice(1) + "View")["catch"](function () {
      return {
        "default": function _default() {
          return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Not found");
        }
      };
    });
  });
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(react__WEBPACK_IMPORTED_MODULE_0__["Suspense"], {
    fallback: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Loading...")
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(_view__WEBPACK_IMPORTED_MODULE_1__["View"], null, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ResourceView, null)));
};

/* harmony default export */ __webpack_exports__["default"] = (ResourceView);

/***/ }),

/***/ "./resources/js/view/index.js":
/*!************************************!*\
  !*** ./resources/js/view/index.js ***!
  \************************************/
/*! exports provided: View */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var _View__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./View */ "./resources/js/view/View.js");
/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "View", function() { return _View__WEBPACK_IMPORTED_MODULE_0__["default"]; });




/***/ }),

/***/ "./resources/js/views lazy recursive ^\\.\\/.*View$":
/*!***************************************************************!*\
  !*** ./resources/js/views lazy ^\.\/.*View$ namespace object ***!
  \***************************************************************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

var map = {
	"./GrammaireView": [
		"./resources/js/views/GrammaireView.js",
		19
	],
	"./NewsView": [
		"./resources/js/views/NewsView.js",
		20
	],
	"./NousView": [
		"./resources/js/views/NousView.js",
		22
	],
	"./ProjectsView": [
		"./resources/js/views/ProjectsView.js",
		24
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
	return __webpack_require__.e(ids[1]).then(function() {
		return __webpack_require__(id);
	});
}
webpackAsyncContext.keys = function webpackAsyncContextKeys() {
	return Object.keys(map);
};
webpackAsyncContext.id = "./resources/js/views lazy recursive ^\\.\\/.*View$";
module.exports = webpackAsyncContext;

/***/ })

}]);