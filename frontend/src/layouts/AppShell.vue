<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import Button from "primevue/button";
import Dialog from "primevue/dialog";
import markLight from "../assets/brand/mark-light.png";
import { logout } from "../auth/oidc";
import SettingsView from "../settings/SettingsView.vue";
import AccountView from "../views/AccountView.vue";
import ChangePasswordView from "../views/ChangePasswordView.vue";

const { t } = useI18n();
const router = useRouter();
const userOpen = ref(false);
const userWrap = ref<HTMLElement | null>(null);
const settingsMode = ref(false);
const infoVisible = ref(false);
const accountVisible = ref(false);
const passwordVisible = ref(false);
const appVersion = "0.1.0";

const repoUrl = "https://github.com/willie68/arcivio";
const readmeUrl = "https://github.com/willie68/arcivio/blob/main/README.md";

function toggleUserMenu() {
  userOpen.value = !userOpen.value;
}

function closeUserMenu() {
  userOpen.value = false;
}

function toggleSettings() {
  closeUserMenu();
  settingsMode.value = !settingsMode.value;
}

function showClient() {
  settingsMode.value = false;
}

function onDocumentClick(event: MouseEvent) {
  if (!userWrap.value?.contains(event.target as Node)) {
    closeUserMenu();
  }
}

function goAccount() {
  closeUserMenu();
  passwordVisible.value = false;
  accountVisible.value = true;
}

function goPassword() {
  closeUserMenu();
  accountVisible.value = false;
  passwordVisible.value = true;
}

function openHelp() {
  window.open(readmeUrl, "_blank", "noopener,noreferrer");
}

function openInfo() {
  closeUserMenu();
  infoVisible.value = true;
}

async function doLogout() {
  closeUserMenu();
  await logout();
  await router.push("/login");
}

onMounted(() => {
  document.addEventListener("click", onDocumentClick);
});
onUnmounted(() => {
  document.removeEventListener("click", onDocumentClick);
});
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <RouterLink class="brand" to="/" :aria-label="t('brand.name')" @click="showClient">
        <img class="mark" :src="markLight" alt="" />
        <span class="wordmark">{{ t("brand.name") }}</span>
      </RouterLink>
      <div class="actions">
        <Button
          icon="pi pi-cog"
          rounded
          text
          :aria-label="t('shell.settings')"
          :aria-pressed="settingsMode"
          :class="{ 'is-on': settingsMode }"
          v-tooltip.bottom="{ value: t('shell.settings'), showDelay: 0 }"
          @click="toggleSettings"
        />
        <Button
          icon="pi pi-question"
          rounded
          text
          :aria-label="t('shell.help')"
          v-tooltip.bottom="{ value: t('shell.help'), showDelay: 0 }"
          @click="openHelp"
        />
        <div ref="userWrap" class="user-wrap">
          <Button
            icon="pi pi-user"
            rounded
            text
            aria-haspopup="true"
            aria-controls="user-menu"
            :aria-expanded="userOpen"
            :aria-label="t('shell.user')"
            v-tooltip.bottom="{ value: t('shell.user'), showDelay: 0 }"
            @click.stop="toggleUserMenu"
          />
          <div v-if="userOpen" id="user-menu" class="user-menu" role="menu">
            <button type="button" role="menuitem" @click="goAccount">
              <i class="pi pi-user" aria-hidden="true" />
              {{ t("account.me") }}
            </button>
            <button type="button" role="menuitem" @click="goPassword">
              <i class="pi pi-key" aria-hidden="true" />
              {{ t("account.changePassword") }}
            </button>
            <button type="button" role="menuitem" @click="openInfo">
              <i class="pi pi-info-circle" aria-hidden="true" />
              {{ t("account.info") }}
            </button>
            <hr />
            <button type="button" role="menuitem" @click="doLogout">
              <i class="pi pi-sign-out" aria-hidden="true" />
              {{ t("account.logout") }}
            </button>
          </div>
        </div>
      </div>
    </header>
    <main class="content" :class="settingsMode ? 'content--settings' : 'content--client'">
      <div v-show="settingsMode" class="mode-host" :aria-hidden="!settingsMode">
        <SettingsView />
      </div>
      <div v-show="!settingsMode" class="mode-host" :aria-hidden="settingsMode">
        <RouterView />
      </div>
      <Dialog
        v-model:visible="passwordVisible"
        modal
        appendTo="self"
        :header="t('account.changePassword')"
        :style="{ width: '24rem' }"
        :draggable="false"
        :dismissableMask="true"
      >
        <ChangePasswordView v-if="passwordVisible" />
      </Dialog>
    </main>
    <footer class="footer">
      <a class="copy" :href="readmeUrl" target="_blank" rel="noopener noreferrer">{{ t("shell.copyright") }}</a>
      <a class="repo" :href="repoUrl" target="_blank" rel="noopener noreferrer">{{ t("shell.github") }}</a>
    </footer>
    <Dialog v-model:visible="accountVisible" modal :header="t('account.me')" :style="{ width: '22rem' }">
      <AccountView v-if="accountVisible" />
    </Dialog>
    <Dialog v-model:visible="infoVisible" modal :header="t('account.info')" :style="{ width: '22rem' }">
      <p class="info-title">{{ t("brand.name") }}</p>
      <p>{{ t("home.tagline") }}</p>
      <p>{{ t("account.version", { version: appVersion }) }}</p>
    </Dialog>
  </div>
</template>

<style scoped>
.shell {
  height: 100vh;
  min-height: 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  overflow: hidden;
}
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.55rem 1rem;
  background: #fff;
  border-bottom: 1px solid #e4e8ec;
}
.brand {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  color: inherit;
  text-decoration: none;
}
.mark {
  width: 2rem;
  height: 2rem;
  object-fit: contain;
}
.wordmark {
  font-size: 1.15rem;
  font-weight: 650;
  letter-spacing: -0.02em;
  color: #1b1f24;
}
.actions {
  display: flex;
  align-items: center;
  gap: 0.15rem;
}
.actions :deep(.p-button) {
  color: #1f4b99;
}
.actions :deep(.p-button:hover) {
  background: #eef3fb;
}
.actions :deep(.p-button.is-on),
.actions :deep(.p-button.is-on:hover) {
  background: #1f4b99;
  color: #fff;
}
.content {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}
.mode-host {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
}
.content :deep(.p-dialog-mask) {
  position: absolute !important;
}
.footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  height: 2.25rem;
  padding: 0 1rem;
  background: #fff;
  border-top: 1px solid #e4e8ec;
  font-size: 0.8rem;
  line-height: 1;
  white-space: nowrap;
}
.footer a {
  color: #5b6570;
  text-decoration: none;
}
.footer a:hover {
  color: #1f4b99;
  text-decoration: underline;
}
.copy {
  overflow: hidden;
  text-overflow: ellipsis;
}
.repo {
  flex-shrink: 0;
  margin-left: auto;
}
.info-title {
  margin: 0 0 0.35rem;
  font-weight: 650;
}
.user-wrap {
  position: relative;
}
.user-menu {
  position: absolute;
  top: calc(100% + 0.35rem);
  right: 0;
  z-index: 20;
  min-width: 13.5rem;
  padding: 0.35rem 0;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 10px 30px rgb(0 0 0 / 12%);
}
.user-menu button {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  width: 100%;
  margin: 0;
  padding: 0.55rem 0.9rem;
  border: 0;
  background: transparent;
  color: #1b1f24;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.user-menu button:hover {
  background: #f4f6f8;
}
.user-menu i {
  color: #5b6570;
}
.user-menu hr {
  margin: 0.3rem 0;
  border: 0;
  border-top: 1px solid #e4e8ec;
}
</style>
