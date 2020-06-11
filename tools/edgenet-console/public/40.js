(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[40],{

/***/ "./resources/js/data/util/session.js":
/*!*******************************************!*\
  !*** ./resources/js/data/util/session.js ***!
  \*******************************************/
/*! exports provided: hash, getSession, setSession, clearSession */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "hash", function() { return hash; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "getSession", function() { return getSession; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "setSession", function() { return setSession; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "clearSession", function() { return clearSession; });
function hash(key) {
  var hash = 0,
      i,
      chr;

  for (i = 0; i < key.length; i++) {
    chr = key.charCodeAt(i);
    hash = (hash << 5) - hash + chr;
    hash |= 0; // Convert to 32bit integer
  }

  return hash;
}

function getSession(hash, key) {
  try {
    return JSON.parse(localStorage.getItem(key + '.' + hash));
  } catch (SyntaxError) {
    return null;
  }
}

function setSession(hash, key, value) {
  localStorage.setItem(key + '.' + hash, JSON.stringify(value));
}

function clearSession(hash) {
  var key = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : null;
  key ? localStorage.removeItem(key + '.' + hash) : localStorage.clear();
}



/***/ })

}]);