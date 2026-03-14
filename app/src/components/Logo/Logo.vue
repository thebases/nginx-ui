<script setup lang="ts">
import darkLogo from '@/assets/img/logo_dark.svg?url'
import lightLogo from '@/assets/img/logo_light.svg?url'
import { useSettingsStore } from '@/pinia'

const settings = useSettingsStore()
const logo = computed(() => settings.theme === 'dark' ? darkLogo : lightLogo)
</script>

<template>
  <div class="logo">
    <img
      :src="logo"
      alt="UI logo"
    >
    <!-- <p class="text">
      Nginx UI
    </p> -->
  </div>
</template>

<style lang="less" scoped>
.dark {
  .logo {
    background-color: transparent;
    -webkit-box-shadow: 1px 1px 0 0 #404040;
    box-shadow: 1px 1px 0 0 #404040;
  }
}

.logo {
  -webkit-box-shadow: 1px 1px 0 0 #e8e8e8;
  box-shadow: 1px 1px 0 0 #e8e8e8;
  transition: all 0.3s;
  height: 64px;
  width: 100%;
  overflow: hidden;
  background-color: #ffffff;

  img {
    height: 46px;
  }

  p.text {
    margin: 0;
    font-size: 22px;
    line-height: 48px;
    height: 48px;
  }
}
</style>
