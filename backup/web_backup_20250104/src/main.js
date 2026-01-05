// import Vue from "vue";

// import VueCompositionAPI from "@vue/composition-api"; // ← 新增

// Vue.use(VueCompositionAPI); // ← 新增

// import Cookies from "js-cookie";

// import "normalize.css/normalize.css";

// import Element from "element-ui";
// import "./styles/element-variables.scss";
// import enLang from "element-ui/lib/locale/lang/en";

// import "@/styles/index.scss";
// import App from "./App";
// import store from "./store";
// import router from "./router";

// import "./icons";
// import "./permission";
// import "./utils/error-log";
// import * as filters from "./filters";
// import "@fortawesome/fontawesome-free/css/all.min.css";

// if (process.env.NODE_ENV === "production") {
//   const { mockXHR } = require("../mock");
//   mockXHR();
// }

// Vue.use(Element, {
//   size: Cookies.get("size") || "medium",
//   locale: enLang,
// });

// Object.keys(filters).forEach((key) => {
//   Vue.filter(key, filters[key]);
// });

// Vue.config.productionTip = false;

// new Vue({
//   el: "#app",
//   router,
//   store,
//   render: (h) => h(App),
// });

import Vue from 'vue'
import VueCompositionAPI from '@vue/composition-api'

Vue.use(VueCompositionAPI)

import Cookies from 'js-cookie'

import 'normalize.css/normalize.css'

import Element from 'element-ui'
import './styles/element-variables.scss'
import enLang from 'element-ui/lib/locale/lang/en'

import '@/styles/index.scss'
import App from './App'
import store from './store'
import router from './router'

import './icons'
import './permission'
import './utils/error-log'
import * as filters from './filters'
import '@fortawesome/fontawesome-free/css/all.min.css'

// 只在生产使用 mock（如果你确实需要）
if (process.env.NODE_ENV === 'production') {
  const { mockXHR } = require('../mock')
  mockXHR()
}

Vue.use(Element, {
  size: Cookies.get('size') || 'medium',
  locale: enLang
})

Object.keys(filters).forEach((key) => {
  Vue.filter(key, filters[key])
})

Vue.config.productionTip = false

new Vue({
  el: '#app',
  router,
  store,
  render: (h) => h(App)
})
