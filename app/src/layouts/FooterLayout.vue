<script setup lang="ts">
const thisYear = computed(() => new Date().getFullYear())
</script>

<template>
  <div class="footer center">
    Copyright © {{ thisYear }} The Base (a fork of NGINX UI)
  </div>
</template>

<style scoped>

</style>
