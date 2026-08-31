<script setup lang="ts">
import { computed } from "vue";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemFooter,
  ItemSeparator,
  ItemTitle,
} from "@/components/ui/item";
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
  <Item variant="outline" class="bg-muted/50">
    <ItemContent>
      <ItemTitle>
        <Label for="ssh-enabled">SSH tunnel</Label>
      </ItemTitle>
    </ItemContent>
    <ItemActions>
      <Switch id="ssh-enabled" :model-value="enabled" @update:model-value="enabled = Boolean($event)" />
    </ItemActions>

    <template v-if="enabled">
      <div class="w-full">
        <SshHopFields :params="params" :is-edit="isEdit" prefix="ssh" host-placeholder="db.internal" />
      </div>
      <ItemSeparator />
      <ItemFooter>
        <Label for="ssh-jump-enabled">Jump host</Label>
        <Switch
          id="ssh-jump-enabled"
          :model-value="jumpEnabled"
          @update:model-value="jumpEnabled = Boolean($event)"
        />
      </ItemFooter>
      <div v-if="jumpEnabled" class="w-full">
        <SshHopFields
          :params="params"
          :is-edit="isEdit"
          prefix="sshJump"
          host-placeholder="jump.example.com"
        />
      </div>
    </template>
  </Item>
</template>
