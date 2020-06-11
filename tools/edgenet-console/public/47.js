(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[47],{

/***/ "./resources/js/form/old/FormFieldSelect.js":
/*!**************************************************!*\
  !*** ./resources/js/form/old/FormFieldSelect.js ***!
  \**************************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! prop-types */ "./node_modules/prop-types/index.js");
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_1___default = /*#__PURE__*/__webpack_require__.n(prop_types__WEBPACK_IMPORTED_MODULE_1__);
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! axios */ "./node_modules/axios/index.js");
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2___default = /*#__PURE__*/__webpack_require__.n(axios__WEBPACK_IMPORTED_MODULE_2__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var grommet_icons__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! grommet-icons */ "./node_modules/grommet-icons/es6/index.js");
function _typeof(obj) { "@babel/helpers - typeof"; if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof = function _typeof(obj) { return typeof obj; }; } else { _typeof = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof(obj); }

function _extends() { _extends = Object.assign || function (target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i]; for (var key in source) { if (Object.prototype.hasOwnProperty.call(source, key)) { target[key] = source[key]; } } } return target; }; return _extends.apply(this, arguments); }

function _objectWithoutProperties(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }

function _objectWithoutPropertiesLoose(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }

function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); if (enumerableOnly) symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; }); keys.push.apply(keys, symbols); } return keys; }

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; if (i % 2) { ownKeys(Object(source), true).forEach(function (key) { _defineProperty(target, key, source[key]); }); } else if (Object.getOwnPropertyDescriptors) { Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)); } else { ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }

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








var MultipleSelectedValue = function MultipleSelectedValue(_ref) {
  var value = _ref.value,
      labelKey = _ref.labelKey,
      onRemove = _ref.onRemove;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
    onClick: function onClick(event) {
      event.preventDefault();
      event.stopPropagation();
      onRemove(value);
    },
    onFocus: function onFocus(event) {
      return event.stopPropagation();
    },
    plain: true
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
    align: "center",
    direction: "row",
    gap: "xsmall",
    pad: {
      vertical: "xsmall",
      horizontal: "small"
    },
    margin: "xsmall",
    background: "neutral-3",
    round: "large"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Text"], {
    size: "small",
    color: "white"
  }, value[labelKey]), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
    background: "white",
    round: "full",
    margin: {
      left: "xsmall"
    }
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_4__["FormClose"], {
    color: "brand",
    size: "small"
  }))));
};

var SelectedValue = function SelectedValue(_ref2) {
  var value = _ref2.value,
      labelKey = _ref2.labelKey,
      onRemove = _ref2.onRemove;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
    direction: "row",
    align: "start",
    justify: "start",
    alignContent: "start"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
    onClick: function onClick(event) {
      event.preventDefault();
      event.stopPropagation();
      onRemove(value);
    },
    onFocus: function onFocus(event) {
      return event.stopPropagation();
    }
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_4__["FormClose"], null)), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
    pad: {
      horizontal: 'xsmall'
    },
    align: "start",
    justify: "start",
    alignContent: "start"
  }, value['secondary'] && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Text"], {
    size: "small"
  }, value['secondary']), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Text"], null, value[labelKey])));
};

var Placeholder = function Placeholder(_ref3) {
  var placeholder = _ref3.placeholder;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
    pad: {
      vertical: "xsmall",
      horizontal: "small"
    },
    margin: "xsmall"
  }, placeholder, "\xA0");
};

var Select = /*#__PURE__*/function (_React$Component) {
  _inherits(Select, _React$Component);

  var _super = _createSuper(Select);

  function Select(props) {
    var _this;

    _classCallCheck(this, Select);

    _this = _super.call(this, props);
    _this.state = {
      options: [],
      count: 0,
      total: 0,
      // holds the filtered options
      // when searching
      searchText: '',
      search: [],
      // mixed value (array or value)
      selected: ''
    };
    _this.selectValue = _this.selectValue.bind(_assertThisInitialized(_this));
    _this.removeValue = _this.removeValue.bind(_assertThisInitialized(_this));
    _this.loadOptions = _this.loadOptions.bind(_assertThisInitialized(_this));
    _this.search = _this.search.bind(_assertThisInitialized(_this));
    _this.localSearch = _this.localSearch.bind(_assertThisInitialized(_this));
    _this.remoteSearch = _this.remoteSearch.bind(_assertThisInitialized(_this));
    return _this;
  }

  _createClass(Select, [{
    key: "componentDidMount",
    value: function componentDidMount() {
      this.loadOptions();
    }
  }, {
    key: "loadOptions",
    value: function loadOptions() {
      var _this2 = this;

      var _this$props = this.props,
          source = _this$props.source,
          limit = _this$props.limit,
          labelKey = _this$props.labelKey,
          valueKey = _this$props.valueKey,
          secondaryKey = _this$props.secondaryKey;
      var _this$state = this.state,
          options = _this$state.options,
          count = _this$state.count,
          total = _this$state.total,
          searchText = _this$state.searchText;
      var offset = options.length;

      if (offset < 0) {
        offset = 0;
      }

      if (offset > 0 && offset >= count) {
        this.setState({
          more: false
        });
        return;
      }

      var params = {};

      if (searchText) {
        params = {
          search: searchText
        };
      }

      if (limit) {
        params = _objectSpread({
          offset: offset,
          limit: limit
        }, params);
      }

      if (secondaryKey) {
        params = _objectSpread({
          fields: [labelKey, valueKey, secondaryKey]
        }, params);
      } else {
        params = _objectSpread({
          fields: [labelKey, valueKey]
        }, params);
      }

      axios__WEBPACK_IMPORTED_MODULE_2___default.a.get(source, {
        params: params
      }).then(function (response) {
        var _response$data = response.data,
            data = _response$data.data,
            meta = _response$data.meta;
        var result = data.map(function (s) {
          var _ref4;

          return _ref4 = {}, _defineProperty(_ref4, valueKey, s[valueKey]), _defineProperty(_ref4, labelKey, s[labelKey]), _defineProperty(_ref4, 'secondary', secondaryKey in s ? s[secondaryKey] : null), _ref4;
        });

        _this2.setState({
          options: limit ? options.concat(result) : result,
          total: meta.total,
          count: meta.count //itemsLoading: false,
          // more: (meta.total > 0)

        });
      });
    }
  }, {
    key: "removeValue",
    value: function removeValue(value) {
      var _this$props2 = this.props,
          name = _this$props2.name,
          valueKey = _this$props2.valueKey,
          multiple = _this$props2.multiple;
      var selected = this.state.selected;
      var update = this.context.update;
      this.setState({
        selected: multiple ? selected.splice(selected.findIndex(function (s) {
          return s[valueKey] === value[valueKey];
        }), 1) : ''
      }, update(name, multiple ? this.state.selected : ''));
    }
  }, {
    key: "selectValue",
    value: function selectValue(_ref5) {
      var value = _ref5.value;
      this.setState({
        selected: value
      });
    }
  }, {
    key: "search",
    value: function search(text) {
      var limit = this.props.limit;
      limit ? this.remoteSearch(text) : this.localSearch(text);
    }
  }, {
    key: "remoteSearch",
    value: function remoteSearch(text) {
      this.setState({
        searchText: text,
        options: [],
        total: 0,
        count: 0
      }, this.loadOptions);
    }
  }, {
    key: "localSearch",
    value: function localSearch(text) {
      var options = this.state.options;
      var labelKey = this.props.labelKey; // search within local options
      // The line below escapes regular expression special characters:
      // [ \ ^ $ . | ? * + ( )

      var escapedText = text.replace(/[-\\^$*+?.()|[\]{}]/g, "\\$&"); // Create the regular expression with modified value which
      // handles escaping special characters. Without escaping special
      // characters, errors will appear in the console

      var exp = new RegExp(escapedText, "i");
      this.setState({
        search: text ? options.filter(function (o) {
          return exp.test(o[labelKey]);
        }) : []
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this3 = this;

      var _this$props3 = this.props,
          name = _this$props3.name,
          label = _this$props3.label,
          placeholder = _this$props3.placeholder,
          labelKey = _this$props3.labelKey,
          valueKey = _this$props3.valueKey,
          secondaryKey = _this$props3.secondaryKey,
          multiple = _this$props3.multiple,
          props = _objectWithoutProperties(_this$props3, ["name", "label", "placeholder", "labelKey", "valueKey", "secondaryKey", "multiple"]);

      var _this$state2 = this.state,
          options = _this$state2.options,
          selected = _this$state2.selected,
          search = _this$state2.search;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Select"], _extends({
        name: name,
        label: label,
        labelKey: labelKey,
        options: search.length ? search : options,
        onMore: this.loadOptions,
        value: selected || '',
        valueKey: valueKey,
        valueLabel: multiple ? /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
          wrap: true,
          direction: "row"
        }, selected && selected.length ? selected.map(function (v) {
          return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(MultipleSelectedValue, {
            key: 'opt' + v[valueKey],
            labelKey: labelKey,
            value: v,
            onRemove: _this3.removeValue
          });
        }) : /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Placeholder, {
          placeholder: placeholder
        })) : selected ? /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(SelectedValue, {
          value: selected,
          labelKey: labelKey,
          onRemove: this.removeValue
        }) : /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Placeholder, {
          placeholder: placeholder
        }),
        onSearch: this.search,
        multiple: multiple,
        onChange: this.selectValue,
        emptySearchMessage: "",
        searchPlaceholder: ""
      }, props), function (option, index, options, state) {
        return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Box"], {
          pad: "small",
          background: state.selected ? 'neutral-3' : ''
        }, option['secondary'] && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["Text"], {
          size: "small"
        }, option['secondary']), option[labelKey]);
      });
    }
  }], [{
    key: "getDerivedStateFromProps",
    value: function getDerivedStateFromProps(props, state) {
      if (props.value !== state.selected) {
        return {
          selected: props.value
        };
      }

      return null;
    }
  }]);

  return Select;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

Select.contextType = grommet__WEBPACK_IMPORTED_MODULE_3__["FormContext"];
Select.defaultProps = {
  labelKey: 'name',
  valueKey: 'id'
};

var FormFieldSelect = function FormFieldSelect(props) {
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_3__["FormField"], _extends({}, props, {
    component: Select,
    plain: false
  }));
};

/* harmony default export */ __webpack_exports__["default"] = (FormFieldSelect);

/***/ })

}]);