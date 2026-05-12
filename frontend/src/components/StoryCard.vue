<script setup lang="ts">
import { useRouter } from 'vue-router'
import type { Story } from '../api/stories'

const props = defineProps<{ story: Story }>()
const emit = defineEmits<{ like: [story: Story] }>()
const router = useRouter()

function openStory() {
  router.push(`/stories/${props.story.slug}`)
}
</script>

<template>
  <article class="story-card panel">
    <div
      class="card-main"
      role="button"
      tabindex="0"
      @click="openStory"
      @keydown.enter.prevent="openStory"
      @keydown.space.prevent="openStory"
    >
      <div class="cover">
        <img v-if="story.coverUrl" :src="story.coverUrl" :alt="story.title" />
        <span v-else>{{ story.title.slice(0, 1) }}</span>
      </div>
      <button class="like-button" type="button" :class="{ liked: story.liked }" @click.stop="emit('like', story)">
        <span v-if="story.liked">&#9829;</span>
        <span v-else>&#9825;</span>
        {{ story.likeCount || 0 }}
      </button>
      <div class="body">
        <h2>{{ story.title }}</h2>
        <p class="description">{{ story.description }}</p>
      </div>
    </div>
  </article>
</template>

<style scoped>
.story-card {
  position: relative;
  overflow: hidden;
  transition: transform 0.18s ease, border-color 0.18s ease;
  border-radius: 8px;
  background: var(--panel-elevated);
}
.story-card:hover {
  transform: translateY(-3px);
  border-color: var(--accent-border);
}
.card-main {
  position: relative;
  display: block;
  width: 100%;
  padding: 0 0 132px;
  text-align: left;
  background: transparent;
  color: inherit;
  cursor: pointer;
  outline: none;
}
.card-main:focus-visible {
  box-shadow: inset 0 0 0 2px var(--accent-border);
}
.cover {
  aspect-ratio: 16 / 10;
  min-height: 0;
  width: 100%;
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, var(--accent-soft), var(--control-strong));
  color: var(--accent-strong);
  font-size: 64px;
  font-weight: 700;
  overflow: hidden;
}
.cover img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.like-button {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 3;
  border-radius: 999px;
  padding: 8px 11px;
  background: color-mix(in srgb, var(--panel-solid) 82%, transparent);
  color: var(--text-strong);
  font-weight: 700;
  box-shadow: 0 8px 24px color-mix(in srgb, var(--bg-deep) 54%, transparent);
  cursor: pointer;
}
.like-button.liked {
  background: var(--accent);
  color: var(--accent-contrast);
}
.body {
  position: absolute;
  z-index: 2;
  left: 0;
  right: 0;
  bottom: 0;
  height: 132px;
  width: 100%;
  padding: 16px 18px 18px;
  display: grid;
  align-content: start;
  gap: 8px;
  background: var(--panel-solid);
  border-top: 1px solid var(--control-hover);
  overflow: hidden;
  transition: height 0.2s ease, background 0.2s ease;
}
.story-card:hover .body {
  height: min(68%, 260px);
  background: var(--panel-elevated);
}
h2 { margin: 0; font-size: 20px; }
.description {
  margin: 0;
  color: var(--muted-strong);
  line-height: 1.55;
  max-height: 74px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
}
.story-card:hover .description {
  display: block;
  -webkit-line-clamp: unset;
  -webkit-box-orient: initial;
  max-height: none;
}
</style>
