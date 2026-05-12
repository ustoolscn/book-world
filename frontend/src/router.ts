import { createRouter, createWebHashHistory } from 'vue-router'
import EnterPage from './pages/EnterPage.vue'
import StoryListPage from './pages/StoryListPage.vue'
import ChatPage from './pages/ChatPage.vue'
import StorySettingsPage from './pages/StorySettingsPage.vue'
import AdminPage from './pages/AdminPage.vue'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/app' },
    { path: '/app', component: EnterPage },
    { path: '/stories', component: StoryListPage },
    { path: '/admin', component: AdminPage },
    { path: '/stories/:slug/settings', component: StorySettingsPage },
    { path: '/stories/:slug', component: ChatPage }
  ]
})
