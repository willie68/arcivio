import { createRouter, createWebHistory } from "vue-router";
import AppShell from "./layouts/AppShell.vue";
import HomeView from "./views/HomeView.vue";
import LoginView from "./views/LoginView.vue";
import CallbackView from "./views/CallbackView.vue";
import { getAccessToken } from "./auth/oidc";

const router = createRouter({
  history: createWebHistory("/"),
  routes: [
    {
      path: "/",
      component: AppShell,
      meta: { auth: true },
      children: [{ path: "", component: HomeView }],
    },
    { path: "/login", component: LoginView },
    { path: "/callback", component: CallbackView },
    { path: "/account", redirect: "/" },
    { path: "/account/password", redirect: "/" },
    { path: "/change-password", redirect: "/" },
  ],
});

router.beforeEach((to) => {
  if (to.matched.some((record) => record.meta.auth) && !getAccessToken()) {
    return { path: "/login" };
  }
  return true;
});

export default router;
