<script setup lang="ts">
import { computed } from "vue";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SSH_AUTH_AGENT, SSH_AUTH_KEY, SSH_AUTH_PASSWORD } from "@/features/connections/ssh";

const props = defineProps<{
  params: Record<string, string>;
  isEdit: boolean;
  prefix: string;
  hostPlaceholder: string;
}>();

function key(suffix: string) {
  return `${props.prefix}${suffix}`;
}

const auth = computed(() => props.params[key("Auth")] || SSH_AUTH_KEY);

function onAuth(value: unknown) {
  props.params[key("Auth")] = String(value ?? SSH_AUTH_KEY);
}
</script>

<template>
  <div class="grid gap-3">
    <div class="grid grid-cols-2 gap-2">
      <div class="grid min-w-0 gap-1.5">
        <Label :for="`${prefix}-host`">Hostname</Label>
        <Input
          :id="`${prefix}-host`"
          v-model="params[key('Host')]"
          :placeholder="hostPlaceholder"
          required
          autocomplete="off"
        />
      </div>
      <div class="grid min-w-0 gap-1.5">
        <Label :for="`${prefix}-port`">Port</Label>
        <Input
          :id="`${prefix}-port`"
          v-model="params[key('Port')]"
          type="number"
          placeholder="22"
          required
          autocomplete="off"
        />
      </div>
    </div>

    <div class="grid min-w-0 gap-1.5">
      <Label :for="`${prefix}-auth`">SSH auth method</Label>
      <Select :model-value="auth" @update:model-value="onAuth">
        <SelectTrigger :id="`${prefix}-auth`" class="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem :value="SSH_AUTH_KEY">Key file</SelectItem>
          <SelectItem :value="SSH_AUTH_PASSWORD">Username & password</SelectItem>
          <SelectItem :value="SSH_AUTH_AGENT">SSH agent</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <template v-if="auth === SSH_AUTH_KEY">
      <div class="grid min-w-0 gap-1.5">
        <Label :for="`${prefix}-user`">SSH username</Label>
        <Input
          :id="`${prefix}-user`"
          v-model="params[key('User')]"
          placeholder="ubuntu"
          required
          autocomplete="off"
        />
      </div>
      <div class="grid grid-cols-2 gap-2">
        <div class="grid min-w-0 gap-1.5">
          <Label :for="`${prefix}-key`">Private key file</Label>
          <Input
            :id="`${prefix}-key`"
            v-model="params[key('KeyPath')]"
            placeholder="~/.ssh/id_ed25519"
            required
            autocomplete="off"
          />
        </div>
        <div class="grid min-w-0 gap-1.5">
          <Label :for="`${prefix}-passphrase`">Passphrase</Label>
          <Input
            :id="`${prefix}-passphrase`"
            v-model="params[key('Passphrase')]"
            type="password"
            placeholder="optional"
            autocomplete="off"
          />
        </div>
      </div>
    </template>

    <div v-else-if="auth === SSH_AUTH_PASSWORD" class="grid grid-cols-2 gap-2">
      <div class="grid min-w-0 gap-1.5">
        <Label :for="`${prefix}-user`">SSH username</Label>
        <Input
          :id="`${prefix}-user`"
          v-model="params[key('User')]"
          placeholder="ubuntu"
          required
          autocomplete="off"
        />
      </div>
      <div class="grid min-w-0 gap-1.5">
        <Label :for="`${prefix}-password`">Password</Label>
        <Input
          :id="`${prefix}-password`"
          v-model="params[key('Password')]"
          type="password"
          :placeholder="isEdit ? 'stored in keychain' : ''"
          :required="!isEdit"
          autocomplete="off"
        />
      </div>
    </div>

    <div v-else class="grid min-w-0 gap-1.5">
      <Label :for="`${prefix}-user`">SSH username</Label>
      <Input
        :id="`${prefix}-user`"
        v-model="params[key('User')]"
        placeholder="ubuntu"
        required
        autocomplete="off"
      />
    </div>
  </div>
</template>
