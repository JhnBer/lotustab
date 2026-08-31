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
import { Switch } from "@/components/ui/switch";
import { SSH_AUTH_AGENT, SSH_AUTH_KEY, SSH_AUTH_PASSWORD } from "@/features/connections/ssh";

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

const auth = computed(() => props.params.sshAuth || SSH_AUTH_KEY);

function onAuth(value: unknown) {
  props.params.sshAuth = String(value ?? SSH_AUTH_KEY);
}
</script>

<template>
  <div class="grid gap-3">
    <div class="flex items-center justify-between gap-4">
      <Label for="ssh-enabled">SSH tunnel</Label>
      <Switch id="ssh-enabled" :model-value="enabled" @update:model-value="enabled = Boolean($event)" />
    </div>

    <template v-if="enabled">
      <div class="grid grid-cols-2 gap-2">
        <div class="grid min-w-0 gap-1.5">
          <Label for="ssh-host">Hostname</Label>
          <Input
            id="ssh-host"
            v-model="params.sshHost"
            placeholder="bastion.example.com"
            required
            autocomplete="off"
          />
        </div>
        <div class="grid min-w-0 gap-1.5">
          <Label for="ssh-port">Port</Label>
          <Input
            id="ssh-port"
            v-model="params.sshPort"
            type="number"
            placeholder="22"
            required
            autocomplete="off"
          />
        </div>
      </div>

      <div class="grid min-w-0 gap-1.5">
        <Label for="ssh-auth">SSH auth method</Label>
        <Select :model-value="auth" @update:model-value="onAuth">
          <SelectTrigger id="ssh-auth" class="w-full">
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
          <Label for="ssh-user">SSH username</Label>
          <Input id="ssh-user" v-model="params.sshUser" placeholder="ubuntu" required autocomplete="off" />
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div class="grid min-w-0 gap-1.5">
            <Label for="ssh-key">Private key file</Label>
            <Input
              id="ssh-key"
              v-model="params.sshKeyPath"
              placeholder="~/.ssh/id_ed25519"
              required
              autocomplete="off"
            />
          </div>
          <div class="grid min-w-0 gap-1.5">
            <Label for="ssh-passphrase">Passphrase</Label>
            <Input
              id="ssh-passphrase"
              v-model="params.sshPassphrase"
              type="password"
              placeholder="optional"
              autocomplete="off"
            />
          </div>
        </div>
      </template>

      <div v-else-if="auth === SSH_AUTH_PASSWORD" class="grid grid-cols-2 gap-2">
        <div class="grid min-w-0 gap-1.5">
          <Label for="ssh-user">SSH username</Label>
          <Input id="ssh-user" v-model="params.sshUser" placeholder="ubuntu" required autocomplete="off" />
        </div>
        <div class="grid min-w-0 gap-1.5">
          <Label for="ssh-password">Password</Label>
          <Input
            id="ssh-password"
            v-model="params.sshPassword"
            type="password"
            :placeholder="isEdit ? 'stored in keychain' : ''"
            :required="!isEdit"
            autocomplete="off"
          />
        </div>
      </div>

      <div v-else class="grid min-w-0 gap-1.5">
        <Label for="ssh-user">SSH username</Label>
        <Input id="ssh-user" v-model="params.sshUser" placeholder="ubuntu" required autocomplete="off" />
      </div>
    </template>
  </div>
</template>
