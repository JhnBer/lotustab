<script setup lang="ts">
import { computed } from "vue";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import SshHopFields from "@/features/connections/SshHopFields.vue";

const props = defineProps<{
  params: Record<string, string>;
  isEdit: boolean;
}>();

const enabled = computed({
  get: () => props.params.sshEnabled === "true",
  set: (value: boolean) => {
    props.params.sshEnabled = value ? "true" : "false";
  },
});

const jumpEnabled = computed({
  get: () => props.params.sshJumpEnabled === "true",
  set: (value: boolean) => {
    props.params.sshJumpEnabled = value ? "true" : "false";
  },
});
</script>

<template>
  <div class="grid gap-3">
    <div class="flex items-center justify-between gap-4">
      <Label for="ssh-enabled">SSH tunnel</Label>
      <Switch id="ssh-enabled" :model-value="enabled" @update:model-value="enabled = Boolean($event)" />
    </div>

    <template v-if="enabled">
      <div class="flex items-center justify-between gap-4">
        <Label for="ssh-jump-enabled">Jump host</Label>
        <Switch
          id="ssh-jump-enabled"
          :model-value="jumpEnabled"
          @update:model-value="jumpEnabled = Boolean($event)"
        />
      </div>

      <SshHopFields
        v-if="jumpEnabled"
        :params="params"
        :is-edit="isEdit"
        prefix="sshJump"
        host-placeholder="jump.example.com"
      />

      <SshHopFields :params="params" :is-edit="isEdit" prefix="ssh" host-placeholder="db.internal" />
    </template>
  </div>
</template>
