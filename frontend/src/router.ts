import { createRouter, createWebHistory } from "vue-router";
import HomeView from "./views/HomeView.vue";
import LoginView from "./views/LoginView.vue";
import CallbackView from "./views/CallbackView.vue";
import { getAccessToken } from "./auth/oidc";

const router = createRouter({
  history: createWebHistory("/"),
  routes: [
    { path: "/", component: HomeView, meta: { auth: true } },
    { path: "/login", component: LoginView },
    { path: "/callback", component: CallbackView },
    { path: "/change-password", redirect: "/login" },
  ],
});

router.beforeEach((to) => {
  if (to.meta.auth && !getAccessToken()) {
    return { path: "/login" };
  }
  return true;
});

export default router;
