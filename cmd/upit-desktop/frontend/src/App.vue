<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import {
  CancelManualUpload,
  ChooseManualUploadFile,
  ConfirmClose,
  CopyManualUploadFinalURL,
  CreateInitialConfigurationSet,
  DeleteShortener,
  DeleteUploader,
  LoadGlobalConfigurationEditor,
  LoadRepairDocument,
  LoadShortenerEditor,
  LoadUploaderEditor,
  FileManagerIntegrationState as LoadIntegrationState,
  FileManagerIntegrationAction,
  PrepareManualUploadFile,
  RetryClose,
  RenameShortener,
  RenameUploader,
  SaveGlobalConfiguration,
  SaveRepairDocument,
  SaveShortenerEditor,
  SaveUploaderEditor,
  SetGlobalConfigurationDirty,
  StartManualUpload,
  StartupState,
} from '../bindings/github.com/HungNth/upit/cmd/upit-desktop/desktopservice.js'
import type {
  DesktopStartupState,
  FileManagerIntegrationState,
  GlobalConfigurationDraft,
  GlobalConfigurationEditorState,
  ManualUploadOptions,
  ManualUploadResult,
  ManualUploadSelection,
  RepairDocumentDraft,
  RepairDocumentKind,
  RepairDocumentState,
  ShortenerEditorDraft,
  ShortenerEditorState,
  UploaderEditorDraft,
  UploaderEditorState,
  UploaderExtractorDraft,
} from '../bindings/github.com/HungNth/upit/internal/app/models.js'
type Area = 'manual-upload' | 'global-configuration' | 'uploaders' | 'shorteners' | 'file-manager-integration'
type DirtyAction = 'navigate' | 'refresh'
type ManualUploadProgress = {
  phase: string
  processed: number
  total: number
}


const areas: Array<{ id: Area; label: string; description: string }> = [
  { id: 'manual-upload', label: 'Manual Upload', description: 'Upload one file interactively.' },
  { id: 'global-configuration', label: 'Global Configuration', description: 'Choose defaults and clipboard behavior.' },
  { id: 'uploaders', label: 'Uploaders', description: 'Inspect available upload destinations.' },
  { id: 'shorteners', label: 'Shorteners', description: 'Inspect optional URL shortening destinations.' },
  { id: 'file-manager-integration', label: 'File Manager Integration', description: 'Inspect installed file-manager integration.' },
]

const activeArea = ref<Area>('manual-upload')
const integrationState = ref<FileManagerIntegrationState | null>(null)
const integrationError = ref('')
const integrationLoading = ref(false)
async function loadIntegration() {
  if (integrationLoading.value) return
  integrationLoading.value = true
  integrationError.value = ''
  try { integrationState.value = await LoadIntegrationState() }
  catch (cause) { integrationState.value = null; integrationError.value = errorMessage(cause) }
  finally { integrationLoading.value = false }
}
watch(activeArea, (area) => { if (area === 'file-manager-integration') void loadIntegration() })
const integrationActionLabels: Record<string, string> = {
  repair: 'Repair Integration', 'open-settings': 'Open Keyboard Settings',
  'open-file-manager': 'Open File Manager to Verify', 'prepare-removal': 'Prepare to Remove Upit',
}
async function runIntegrationAction(action: string) {
  if (integrationLoading.value) return
  const confirmed = action !== 'prepare-removal' || window.confirm('Prepare to remove Upit? This unregisters its Finder Service and closes Desktop. Afterwards move Upit.app to Trash. Your Configuration Set is retained.')
  if (!confirmed) return
  integrationLoading.value = true
  integrationError.value = ''
  try { integrationState.value = await FileManagerIntegrationAction(action, confirmed) }
  catch (cause) {
    integrationState.value = null
    integrationError.value = errorMessage(cause)
    try { integrationState.value = await LoadIntegrationState() }
    catch (refreshCause) { integrationError.value += ` Status refresh failed: ${errorMessage(refreshCause)}` }
  }
  finally { integrationLoading.value = false }
}
const activeAreaDetails = computed(() => areas.find((area) => area.id === activeArea.value) ?? areas[0])
const dirtyEditorLabel = computed(() => activeArea.value === 'uploaders' ? 'Uploader' : activeArea.value === 'shorteners' ? 'Shortener' : 'Global Configuration')
const state = ref<DesktopStartupState | null>(null)
const loading = ref(true)
const error = ref('')
const setupLoading = ref(false)
const setupError = ref('')
const setupGlobal = reactive<GlobalConfigurationDraft>({ revision: '', defaultUploader: '', defaultShortener: '', copyToClipboard: false })
const setupUploader = reactive<UploaderEditorDraft>({
	revision: '',
	originalName: '',
	name: '',
	request: { method: '', url: '', headers: [], query: [], body: '', fileField: '', fields: [], dataJSON: '' },
	response: { url: { type: '', path: '', header: '', pattern: '', group: '' }, error: null },
})
type RepairKind = 'config' | 'uploaders' | 'shorteners'
const repairKind = ref<RepairKind>('config')
const repairState = ref<RepairDocumentState | null>(null)
const repairLocked = ref(true)
const repairLoading = ref(false)
const repairError = ref('')
function lockRepair() {
	repairLocked.value = true
	repairState.value = null
}
function changeRepairKind() {
	repairError.value = ''
	lockRepair()
}


function countSetupInputs(value: unknown): number {
	if (value === '{input}') return 1
	if (Array.isArray(value)) return value.reduce((count, item) => count + countSetupInputs(item), 0)
	if (value && typeof value === 'object') return Object.values(value).reduce((count, item) => count + countSetupInputs(item), 0)
	return 0
}

function setupJSONReady(text: string) {
	try {
		const value = JSON.parse(text)
		return Boolean(value && typeof value === 'object' && !Array.isArray(value) && countSetupInputs(value) === 1)
	} catch {
		return false
	}
}

const setupReady = computed(() => {
	const request = setupUploader.request
	const urlReady = /^https?:\/\/[^\s]+$/i.test(request.url)
	const bodyReady = request.body === 'binary'
		? true
		: request.body === 'multipart'
			? Boolean(request.fileField)
			: request.body === 'form'
				? request.fields.some((entry) => Boolean(entry.key) && entry.value === '{input}')
				: request.body === 'json' && setupJSONReady(request.dataJSON)
	const extractor = setupUploader.response.url
	const extractorReady = extractor.type === 'body'
		|| (extractor.type === 'json' && Boolean(extractor.path))
		|| (extractor.type === 'header' && Boolean(extractor.header))
		|| (extractor.type === 'regex' && Boolean(extractor.pattern))
	return Boolean(!setupGlobal.defaultShortener && setupGlobal.defaultUploader && setupUploader.name && request.method && urlReady && bodyReady && extractorReady)
})
const globalEditor = reactive<GlobalConfigurationEditorState>({
  configurationPath: '',
  revision: '',
  defaultUploader: '',
  defaultShortener: '',
  copyToClipboard: false,
  uploaders: [],
  shorteners: [],
  shortenersPresent: false,
})
const globalSaved = ref('')
const globalLoading = ref(false)
const globalError = ref('')
const globalNotice = ref('')
const dirtyAction = ref<DirtyAction | null>(null)
const pendingArea = ref<Area | null>(null)
const closeRequested = ref(false)
let stopCloseRequested: (() => void) | undefined
let stopManualFilesDropped: (() => void) | undefined
let stopManualUploadProgress: (() => void) | undefined
let stopManualUploadCloseRequested: (() => void) | undefined
const globalDraft = computed<GlobalConfigurationDraft>(() => ({
  revision: globalEditor.revision,
  defaultUploader: globalEditor.defaultUploader,
  defaultShortener: globalEditor.defaultShortener,
  copyToClipboard: globalEditor.copyToClipboard,
}))
const globalDirty = computed(() => Boolean(globalEditor.revision) && globalSaved.value !== JSON.stringify(globalDraft.value))
const uploaderState = ref<UploaderEditorState | null>(null)
const uploaderSaved = ref('')
const uploaderLoading = ref(false)
const uploaderError = ref('')
const pendingUploader = ref<string | null>(null)
const revealedSecrets = ref<Record<string, boolean>>({})
const uploaderDirty = computed(() => Boolean(uploaderState.value?.draft.revision) && uploaderSaved.value !== JSON.stringify(uploaderState.value?.draft))
const shortenerState = ref<ShortenerEditorState | null>(null)
const shortenerSaved = ref('')
const shortenerLoading = ref(false)
const shortenerError = ref('')
const pendingShortener = ref<string | null>(null)
const shortenerDirty = computed(() => Boolean(shortenerState.value) && shortenerSaved.value !== JSON.stringify(shortenerState.value?.draft))
const renameDialog = reactive<{
  open: boolean
  kind: 'uploader' | 'shortener'
  currentName: string
  newName: string
  error: string
  loading: boolean
}>({
  open: false,
  kind: 'uploader',
  currentName: '',
  newName: '',
  error: '',
  loading: false,
})
const renameInputRef = ref<HTMLInputElement | null>(null)
const renameDialogRef = ref<HTMLElement | null>(null)
const deleteCancelRef = ref<HTMLButtonElement | null>(null)
const deleteDialogRef = ref<HTMLElement | null>(null)
const dirtyDialogRef = ref<HTMLElement | null>(null)
const dirtyCancelRef = ref<HTMLButtonElement | null>(null)
const manualCloseDialogRef = ref<HTMLElement | null>(null)
const manualCloseCancelRef = ref<HTMLButtonElement | null>(null)
const closeDialogRef = ref<HTMLElement | null>(null)
const closeCancelRef = ref<HTMLButtonElement | null>(null)
let lastFocusedElement: HTMLElement | null = null
let lastModalFocusedElement: HTMLElement | null = null

function trapDialogTab(e: KeyboardEvent, container: HTMLElement | null) {
  if (e.key !== 'Tab' || !container) return
  const focusable = container.querySelectorAll<HTMLElement>(
    'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'
  )
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}

function onRenameDialogKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (!renameDialog.loading) closeRenameDialog()
    return
  }
  trapDialogTab(e, renameDialogRef.value)
}

function onDeleteDialogKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (!deleteDialog.loading) closeDeleteDialog()
    return
  }
  trapDialogTab(e, deleteDialogRef.value)
}

function onDirtyDialogKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    resolveDirtyAction('cancel')
    return
  }
  trapDialogTab(e, dirtyDialogRef.value)
}

function onManualCloseDialogKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    void resolveManualUploadClose('cancel')
    return
  }
  trapDialogTab(e, manualCloseDialogRef.value)
}

function onCloseDialogKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    resolveClose('cancel')
    return
  }
  trapDialogTab(e, closeDialogRef.value)
}

watch(
  dirtyAction,
  async (action) => {
    if (action) {
      lastModalFocusedElement = document.activeElement as HTMLElement | null
      await nextTick()
      dirtyCancelRef.value?.focus()
    } else {
      await nextTick()
      lastModalFocusedElement?.focus()
      lastModalFocusedElement = null
    }
  },
)


watch(
  closeRequested,
  async (requested) => {
    if (requested) {
      lastModalFocusedElement = document.activeElement as HTMLElement | null
      await nextTick()
      closeCancelRef.value?.focus()
    } else {
      await nextTick()
      lastModalFocusedElement?.focus()
      lastModalFocusedElement = null
    }
  },
)

watch(
  () => renameDialog.open,
  async (open) => {
    if (open) {
      lastFocusedElement = document.activeElement as HTMLElement | null
      await nextTick()
      renameInputRef.value?.focus()
      renameInputRef.value?.select()
    } else {
      await nextTick()
      lastFocusedElement?.focus()
      lastFocusedElement = null
    }
  },
)

const deleteDialog = reactive<{
  open: boolean
  kind: 'uploader' | 'shortener'
  name: string
  error: string
  loading: boolean
}>({
  open: false,
  kind: 'uploader',
  name: '',
  error: '',
  loading: false,
})

watch(
  () => deleteDialog.open,
  async (open) => {
    if (open) {
      lastFocusedElement = document.activeElement as HTMLElement | null
      await nextTick()
      deleteCancelRef.value?.focus()
    } else {
      await nextTick()
      lastFocusedElement?.focus()
      lastFocusedElement = null
    }
  },
)
const manualFile = ref<ManualUploadSelection | null>(null)
const manualUploaderChoice = ref('default')
const manualShortenerChoice = ref('default')
const manualClipboard = ref('default')
const manualTimeout = ref('')
const manualLoading = ref(false)
const manualError = ref('')
const manualNotice = ref('')
const manualResult = ref<ManualUploadResult | null>(null)
const manualProgress = ref<ManualUploadProgress | null>(null)
const manualCloseRequested = ref(false)
watch(
  manualCloseRequested,
  async (requested) => {
    if (requested) {
      lastModalFocusedElement = document.activeElement as HTMLElement | null
      await nextTick()
      manualCloseCancelRef.value?.focus()
    } else {
      await nextTick()
      lastModalFocusedElement?.focus()
      lastModalFocusedElement = null
    }
  },
)
const anyDirty = computed(() => globalDirty.value || uploaderDirty.value || shortenerDirty.value)
const manualReady = computed(() => Boolean(
  state.value?.mode === 'normal'
  && manualFile.value
  && !anyDirty.value
  && !manualLoading.value,
))
watch(anyDirty, (dirty) => {
 	void SetGlobalConfigurationDirty(dirty)

})
function errorMessage(cause: unknown) {
  return cause instanceof Error ? cause.message : String(cause)
}

function manualPhaseLabel(phase: string) {
  return ({
    preparing: 'Preparing',
    uploading: 'Uploading',
    response: 'Processing response',
    shortening: 'Shortening URL',
    clipboard: 'Copying to clipboard',
  } as Record<string, string>)[phase] ?? 'Working'
}

function formatManualBytes(value: number) {
  if (value < 1024) return `${value} bytes`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`
  return `${(value / (1024 * 1024)).toFixed(1)} MiB`
}

function applyManualUploadFile(selection: ManualUploadSelection) {
  if (!selection.path) {
    return
  }
  manualFile.value = selection
  manualError.value = ''
  manualNotice.value = ''
  manualResult.value = null
}

function manualUploader(): string {
  return manualUploaderChoice.value.startsWith('named:') ? manualUploaderChoice.value.slice('named:'.length) : ''
}

function manualShortener(): { name: string; disabled: boolean } {
  if (manualShortenerChoice.value === 'none') {
    return { name: '', disabled: true }
  }
  return {
    name: manualShortenerChoice.value.startsWith('named:') ? manualShortenerChoice.value.slice('named:'.length) : '',
    disabled: false,
  }
}

function manualUploadOptions(): ManualUploadOptions | null {
  if (!manualFile.value) {
    return null
  }
  const shortener = manualShortener()
  return {
    filePath: manualFile.value.path,
    uploader: manualUploader(),
    shortener: shortener.name,
    disableShortening: shortener.disabled,
    clipboard: manualClipboard.value,
    timeout: manualTimeout.value,
  }
}

async function prepareManualUploadFile(filePath: string) {
  if (manualLoading.value) {
    return
  }
  try {
    applyManualUploadFile(await PrepareManualUploadFile(filePath))
  } catch (cause) {
    manualError.value = errorMessage(cause)
  }
}

async function chooseManualUploadFile() {
  if (manualLoading.value) {
    return
  }
  try {
    applyManualUploadFile(await ChooseManualUploadFile())
  } catch (cause) {
    manualError.value = errorMessage(cause)
  }
}

async function startManualUpload() {
  const options = manualUploadOptions()
  if (!options || !manualReady.value) {
    return
  }
  manualLoading.value = true
  manualError.value = ''
  manualNotice.value = ''
  manualResult.value = null
  manualProgress.value = { phase: 'preparing', processed: 0, total: 0 }
  try {
    const result = await StartManualUpload(options)
    manualResult.value = result
    if (result.success) {
      manualFile.value = null
    }
  } catch (cause) {
    manualError.value = errorMessage(cause)
  } finally {
    manualLoading.value = false
  }
}

async function copyManualUploadFinalURL() {
  if (!manualResult.value?.success) {
    return
  }
  manualNotice.value = ''
  try {
    await CopyManualUploadFinalURL(manualResult.value.finalURL)
    manualNotice.value = 'Final URL copied to the clipboard.'
  } catch (cause) {
    manualError.value = errorMessage(cause)
  }
}

async function cancelManualUpload() {
  if (!manualLoading.value) {
    return
  }
  if (!await CancelManualUpload()) {
    manualError.value = 'Manual Upload could not be canceled.'
  }
}

async function resolveManualUploadClose(action: 'cancel' | 'confirm') {
  manualCloseRequested.value = false
  if (action === 'cancel') {
    return
  }
  await cancelManualUpload()
  await RetryClose()
}

function applyGlobalEditor(loaded: GlobalConfigurationEditorState) {
  globalEditor.configurationPath = loaded.configurationPath
  globalEditor.revision = loaded.revision
  globalEditor.defaultUploader = loaded.defaultUploader
  globalEditor.defaultShortener = loaded.defaultShortener
  globalEditor.copyToClipboard = loaded.copyToClipboard
  globalEditor.uploaders = loaded.uploaders ?? []
  globalEditor.shorteners = loaded.shorteners ?? []
  globalEditor.shortenersPresent = loaded.shortenersPresent
  if (state.value) {
    state.value.defaultUploader = loaded.defaultUploader
    state.value.defaultShortener = loaded.defaultShortener
    state.value.copyToClipboard = loaded.copyToClipboard
  }
  globalSaved.value = JSON.stringify(globalDraft.value)
}

async function loadGlobalEditor() {
  globalLoading.value = true
  globalError.value = ''
  try {
    applyGlobalEditor(await LoadGlobalConfigurationEditor())
  } catch (cause) {
    globalError.value = errorMessage(cause)
  } finally {
    globalLoading.value = false
  }
}

function applyUploaderEditor(loaded: UploaderEditorState) {
	uploaderState.value = loaded
	uploaderSaved.value = JSON.stringify(loaded.draft)
	revealedSecrets.value = {}
}

async function loadUploaderEditor(name: string) {
	uploaderLoading.value = true
	uploaderError.value = ''
	try {
		applyUploaderEditor(await LoadUploaderEditor(name))
	} catch (cause) {
		uploaderError.value = errorMessage(cause)
	} finally {
		uploaderLoading.value = false
	}
}

async function saveUploaderEditor() {
	if (!uploaderState.value || !uploaderDirty.value) {
		return true
	}
	uploaderLoading.value = true
	uploaderError.value = ''
	try {
		applyUploaderEditor(await SaveUploaderEditor(uploaderState.value.draft))
		return true
	} catch (cause) {
		uploaderError.value = errorMessage(cause)
		return false
	} finally {
		uploaderLoading.value = false
	}
}

function renameUploader() {
	if (!uploaderState.value?.draft.originalName) {
		return
	}
	renameDialog.kind = 'uploader'
	renameDialog.currentName = uploaderState.value.draft.originalName
	renameDialog.newName = uploaderState.value.draft.originalName
	renameDialog.error = ''
	renameDialog.loading = false
	renameDialog.open = true
}

function deleteUploader() {
	if (!uploaderState.value?.draft.originalName) {
		return
	}
	deleteDialog.kind = 'uploader'
	deleteDialog.name = uploaderState.value.draft.originalName
	deleteDialog.error = ''
	deleteDialog.loading = false
	deleteDialog.open = true
}

function applyShortenerEditor(loaded: ShortenerEditorState) {
	shortenerState.value = loaded
	shortenerSaved.value = JSON.stringify(loaded.draft)
}

async function loadShortenerEditor(name: string) {
	shortenerLoading.value = true
	shortenerError.value = ''
	try {
		applyShortenerEditor(await LoadShortenerEditor(name))
	} catch (cause) {
		shortenerError.value = errorMessage(cause)
	} finally {
		shortenerLoading.value = false
	}
}

async function saveShortenerEditor() {
	if (!shortenerState.value || !shortenerDirty.value) {
		return true
	}
	shortenerLoading.value = true
	shortenerError.value = ''
	try {
		applyShortenerEditor(await SaveShortenerEditor(shortenerState.value.draft))
		return true
	} catch (cause) {
		shortenerError.value = errorMessage(cause)
		return false
	} finally {
		shortenerLoading.value = false
	}
}

function newShortener() {
	if (anyDirty.value) {
		pendingShortener.value = ''
		dirtyAction.value = 'navigate'
		return
	}
	void loadShortenerEditor('')
}

function selectShortener(name: string) {
	if (shortenerState.value?.draft.originalName === name) {
		return
	}
	if (anyDirty.value) {
		pendingShortener.value = name
		dirtyAction.value = 'navigate'
		return
	}
	void loadShortenerEditor(name)
}

function renameShortener() {
	if (!shortenerState.value?.draft.originalName) {
		return
	}
	renameDialog.kind = 'shortener'
	renameDialog.currentName = shortenerState.value.draft.originalName
	renameDialog.newName = shortenerState.value.draft.originalName
	renameDialog.error = ''
	renameDialog.loading = false
	renameDialog.open = true
}

function deleteShortener() {
	if (!shortenerState.value?.draft.originalName) {
		return
	}
	deleteDialog.kind = 'shortener'
	deleteDialog.name = shortenerState.value.draft.originalName
	deleteDialog.error = ''
	deleteDialog.loading = false
	deleteDialog.open = true
}

function closeRenameDialog() {
	if (renameDialog.loading) return
	renameDialog.open = false
	renameDialog.error = ''
}

async function submitRename() {
	const newName = renameDialog.newName
	if (!newName.trim()) {
		renameDialog.error = 'Name cannot be empty'
		return
	}
	if (newName === renameDialog.currentName) {
		renameDialog.open = false
		renameDialog.error = ''
		return
	}
	renameDialog.loading = true
	renameDialog.error = ''
	try {
		if (renameDialog.kind === 'uploader') {
			if (!uploaderState.value) return
			uploaderLoading.value = true
			try {
				applyUploaderEditor(await RenameUploader({
					revision: uploaderState.value.revision,
					originalName: renameDialog.currentName,
					newName,
				}))
			} finally {
				uploaderLoading.value = false
			}
		} else {
			if (!shortenerState.value) return
			shortenerLoading.value = true
			try {
				applyShortenerEditor(await RenameShortener({
					revision: shortenerState.value.revision,
					originalName: renameDialog.currentName,
					newName,
				}))
			} finally {
				shortenerLoading.value = false
			}
		}
		renameDialog.open = false
		renameDialog.error = ''
	} catch (cause) {
		renameDialog.error = errorMessage(cause)
	} finally {
		renameDialog.loading = false
	}
}

function closeDeleteDialog() {
	if (deleteDialog.loading) return
	deleteDialog.open = false
	deleteDialog.error = ''
}

async function submitDelete() {
	deleteDialog.loading = true
	deleteDialog.error = ''
	try {
		if (deleteDialog.kind === 'uploader') {
			if (!uploaderState.value) return
			uploaderLoading.value = true
			try {
				applyUploaderEditor(await DeleteUploader({
					revision: uploaderState.value.revision,
					name: deleteDialog.name,
				}))
			} finally {
				uploaderLoading.value = false
			}
		} else {
			if (!shortenerState.value) return
			shortenerLoading.value = true
			try {
				applyShortenerEditor(await DeleteShortener({
					revision: shortenerState.value.revision,
					name: deleteDialog.name,
				}))
			} finally {
				shortenerLoading.value = false
			}
		}
		deleteDialog.open = false
		deleteDialog.error = ''
	} catch (cause) {
		deleteDialog.error = errorMessage(cause)
	} finally {
		deleteDialog.loading = false
	}
}
function selectUploader(name: string) {
	if (uploaderState.value?.draft.originalName === name) {
		return
	}
	if (anyDirty.value) {
		pendingUploader.value = name
		dirtyAction.value = 'navigate'
		return
	}
	void loadUploaderEditor(name)
}

function newUploader() {
	if (anyDirty.value) {
		pendingUploader.value = ''
		dirtyAction.value = 'navigate'
		return
	}
	void loadUploaderEditor('')
}

function addMapEntry(kind: 'headers' | 'query' | 'fields') {
	if (!uploaderState.value) {
		return
	}
	const entries = uploaderState.value.draft.request[kind] ?? []
	entries.push({ key: '', value: '', sensitive: false })
	uploaderState.value.draft.request[kind] = entries
}

function removeMapEntry(kind: 'headers' | 'query' | 'fields', index: number) {
	if (!uploaderState.value) {
		return
	}
	const entries = uploaderState.value.draft.request[kind] ?? []
	entries.splice(index, 1)
	uploaderState.value.draft.request[kind] = entries
}

function addShortenerMapEntry(kind: 'headers' | 'query') {
	if (!shortenerState.value) return
	const entries = shortenerState.value.draft.request[kind] ?? []
	entries.push({ key: '', value: '', sensitive: false })
	shortenerState.value.draft.request[kind] = entries
}

function removeShortenerMapEntry(kind: 'headers' | 'query', index: number) {
	if (!shortenerState.value) return
	const entries = shortenerState.value.draft.request[kind] ?? []
	entries.splice(index, 1)
	shortenerState.value.draft.request[kind] = entries
}

const shortenerMapKinds: Array<{ key: 'headers' | 'query'; label: string }> = [
	{ key: 'headers', label: 'Headers' },
	{ key: 'query', label: 'Query parameters' },
]
function secretKey(kind: string, index: number) {
	return `${kind}-${index}`
}

function toggleSecret(kind: string, index: number) {
	const key = secretKey(kind, index)
	revealedSecrets.value[key] = !revealedSecrets.value[key]
}

const uploaderMapKinds: Array<{ key: 'headers' | 'query' | 'fields'; label: string }> = [
	{ key: 'headers', label: 'Headers' },
	{ key: 'query', label: 'Query parameters' },
	{ key: 'fields', label: 'Form fields' },
]

function addSetupField() {
	setupUploader.request.fields.push({ key: '', value: '', sensitive: false })
}

function removeSetupField(index: number) {
	setupUploader.request.fields.splice(index, 1)
}

function toggleErrorExtractor(enabled: boolean) {
	if (!uploaderState.value) {
		return
	}
	if (enabled && !uploaderState.value.draft.response.error) {
		uploaderState.value.draft.response.error = { type: 'body', path: '', header: '', pattern: '', group: '' }
	} else if (!enabled) {
		uploaderState.value.draft.response.error = null
	}
}

function toggleShortenerError(enabled: boolean) {
	if (!shortenerState.value) return
	if (enabled && !shortenerState.value.draft.response.error) {
		shortenerState.value.draft.response.error = { type: 'json', path: '', header: '', pattern: '', group: '' }
	} else if (!enabled) {
		shortenerState.value.draft.response.error = null
	}
}

function onShortenerErrorToggle(event: Event) {
	toggleShortenerError(event.target instanceof HTMLInputElement && event.target.checked)
}

function resetExtractorFields(extractor: UploaderExtractorDraft, type: string) {
	extractor.type = type
	if (type === 'body') {
		extractor.path = ''
		extractor.header = ''
		extractor.pattern = ''
		extractor.group = ''
	} else if (type === 'json') {
		extractor.header = ''
		extractor.pattern = ''
		extractor.group = ''
	} else if (type === 'header') {
		extractor.path = ''
		extractor.pattern = ''
		extractor.group = ''
	} else if (type === 'regex') {
		extractor.path = ''
		extractor.header = ''
	}
}

function changeUploaderExtractor(event: Event) {
	if (!uploaderState.value) return
	const type = (event.target as HTMLSelectElement).value
	resetExtractorFields(uploaderState.value.draft.response.url, type)
}

function changeUploaderErrorExtractor(event: Event) {
	if (!uploaderState.value?.draft.response.error) return
	const type = (event.target as HTMLSelectElement).value
	resetExtractorFields(uploaderState.value.draft.response.error, type)
}

function changeSetupExtractor(event: Event) {
	const type = (event.target as HTMLSelectElement).value
	resetExtractorFields(setupUploader.response.url, type)
}

function onErrorToggle(event: Event) {
	toggleErrorExtractor(event.target instanceof HTMLInputElement && event.target.checked)
}

function changeUploaderBody(event: Event) {
	if (!uploaderState.value) {
		return
	}
	const body = (event.target as HTMLSelectElement).value as 'multipart' | 'binary' | 'form' | 'json'
	uploaderState.value.draft.request.body = body
	if (body === 'binary') {
		uploaderState.value.draft.request.fileField = ''
		uploaderState.value.draft.request.fields = []
		uploaderState.value.draft.request.dataJSON = ''
	} else if (body === 'form') {
		uploaderState.value.draft.request.fileField = ''
		uploaderState.value.draft.request.dataJSON = ''
	} else if (body === 'json') {
		uploaderState.value.draft.request.fileField = ''
		uploaderState.value.draft.request.fields = []
	} else {
		uploaderState.value.draft.request.dataJSON = ''
	}
}

async function saveGlobalEditor() {
  if (!globalDirty.value) {
    return true
  }
  globalLoading.value = true
  globalError.value = ''
  globalNotice.value = ''
  try {
    applyGlobalEditor(await SaveGlobalConfiguration(globalDraft.value))
    globalNotice.value = 'Global Configuration saved.'
    return true
  } catch (cause) {
    globalError.value = errorMessage(cause)
    return false
  } finally {
    globalLoading.value = false
  }
}

async function finishSetup() {
	setupLoading.value = true
	setupError.value = ''
	try {
		const created = await CreateInitialConfigurationSet(setupGlobal, setupUploader)
		state.value = created
		await loadGlobalEditor()
		await loadUploaderEditor(created.defaultUploader)
		await loadShortenerEditor('')
		activeArea.value = 'manual-upload'
	} catch (cause) {
		setupError.value = errorMessage(cause)
	} finally {
		setupLoading.value = false
	}
}

async function loadRepairDocument() {
	repairLoading.value = true
	repairError.value = ''
	try {
		repairState.value = await LoadRepairDocument(repairKind.value)
	} catch (cause) {
		repairError.value = errorMessage(cause)
	} finally {
		repairLoading.value = false
	}
}

async function unlockRepair() {
	lockRepair()
	await loadRepairDocument()
	if (repairState.value) {
		repairLocked.value = false
	}
}

async function saveRepairDocument() {
	if (!repairState.value) return
	repairLoading.value = true
	repairError.value = ''
	try {
		const startup = await SaveRepairDocument({ kind: repairKind.value, revision: repairState.value.revision, content: repairState.value.content })
		state.value = startup
		lockRepair()
	} catch (cause) {
		repairError.value = errorMessage(cause)
	} finally {
		repairLoading.value = false
	}
}
function requestArea(area: Area) {
  if ((state.value?.mode !== 'normal' && area !== 'file-manager-integration') || area === activeArea.value) {
    return
  }
  if (anyDirty.value) {
    pendingArea.value = area
    dirtyAction.value = 'navigate'
    return
  }
  activeArea.value = area
  if (area === 'uploaders' && !uploaderState.value) {
    void loadUploaderEditor(state.value?.defaultUploader ?? '')
  }
  if (area === 'shorteners' && !shortenerState.value) {
    void loadShortenerEditor(state.value?.defaultShortener ?? '')
  }
}

function requestRefresh() {
  if (anyDirty.value) {
    dirtyAction.value = 'refresh'
    return
  }
  if (activeArea.value === 'uploaders') {
    void loadUploaderEditor(uploaderState.value?.draft.originalName ?? '')
    return
  }
  if (activeArea.value === 'shorteners') {
    void loadShortenerEditor(shortenerState.value?.draft.originalName ?? '')
    return
  }
  void loadGlobalEditor()
}

async function resolveDirtyAction(action: 'save' | 'discard' | 'cancel') {
  if (action === 'cancel') {
    dirtyAction.value = null
    pendingArea.value = null
    pendingUploader.value = null
    pendingShortener.value = null
    return
  }
  const savingUploader = activeArea.value === 'uploaders' && uploaderDirty.value
  const savingShortener = activeArea.value === 'shorteners' && shortenerDirty.value
  if (action === 'save' && !(savingUploader ? await saveUploaderEditor() : savingShortener ? await saveShortenerEditor() : await saveGlobalEditor())) {
    return
  }
  if (action === 'discard') {
    if (savingUploader) {
      await loadUploaderEditor(uploaderState.value?.draft.originalName ?? '')
    } else if (savingShortener) {
      await loadShortenerEditor(shortenerState.value?.draft.originalName ?? '')
    } else {
      await loadGlobalEditor()
    }
  }
  const nextArea = pendingArea.value
  const nextUploader = pendingUploader.value
  const nextShortener = pendingShortener.value
  const nextAction = dirtyAction.value
  dirtyAction.value = null
  pendingArea.value = null
  pendingUploader.value = null
  pendingShortener.value = null
  if (nextAction === 'navigate' && nextArea) {
    activeArea.value = nextArea
  }
  if (nextAction === 'navigate' && nextUploader !== null) {
    await loadUploaderEditor(nextUploader)
  }
  if (nextAction === 'navigate' && nextShortener !== null) {
    await loadShortenerEditor(nextShortener)
  }
}

async function resolveClose(action: 'save' | 'discard' | 'cancel') {
	if (action === 'cancel') {
		closeRequested.value = false
		return
	}
	const savingUploader = activeArea.value === 'uploaders' && uploaderDirty.value
	const savingShortener = activeArea.value === 'shorteners' && shortenerDirty.value
	if (action === 'save' && !(savingUploader ? await saveUploaderEditor() : savingShortener ? await saveShortenerEditor() : await saveGlobalEditor())) {
		return
	}
	if (action === 'discard') {
		if (savingUploader) {
			await loadUploaderEditor(uploaderState.value?.draft.originalName ?? '')
			if (uploaderError.value) return
		} else if (savingShortener) {
			await loadShortenerEditor(shortenerState.value?.draft.originalName ?? '')
			if (shortenerError.value) return
		} else {
			await loadGlobalEditor()
			if (globalError.value) return
		}
	}
	closeRequested.value = false
	await SetGlobalConfigurationDirty(false)
	await ConfirmClose()
}

const handleFocus = () => {
	if (activeArea.value === 'file-manager-integration') { void loadIntegration(); return }
	if (state.value?.mode !== 'normal' || anyDirty.value) return
	if (activeArea.value === 'uploaders') {
		void loadUploaderEditor(uploaderState.value?.draft.originalName ?? '')
		return
	}
	if (activeArea.value === 'shorteners') {
		void loadShortenerEditor(shortenerState.value?.draft.originalName ?? '')
		return
	}
	void loadGlobalEditor()
}

onMounted(async () => {
	try {
		const loaded = await StartupState()
		if (!loaded || typeof loaded.mode !== 'string') throw new Error('Desktop startup state was not returned')
		state.value = loaded
		if (loaded.mode === 'normal') {
			await loadGlobalEditor()
			await loadUploaderEditor(loaded.defaultUploader)
			await loadShortenerEditor(loaded.defaultShortener)
		}
	} catch (cause) {
		error.value = errorMessage(cause)
	} finally {
		loading.value = false
	}
	window.addEventListener('focus', handleFocus)
	stopCloseRequested = Events.On('desktop:close-requested', () => { closeRequested.value = true })
	stopManualFilesDropped = Events.On('desktop:manual-upload-files-dropped', (event) => {
		const files = event.data
		if (!Array.isArray(files) || files.length !== 1 || typeof files[0] !== 'string') {
			manualError.value = 'Drop exactly one file.'
			return
		}
		void prepareManualUploadFile(files[0])
	})
	stopManualUploadProgress = Events.On('desktop:manual-upload-progress', (event) => {
		const update = event.data as Partial<ManualUploadProgress>
		if (typeof update?.phase === 'string' && typeof update.processed === 'number' && typeof update.total === 'number') {
			manualProgress.value = { phase: update.phase, processed: update.processed, total: update.total }
		}
	})
	stopManualUploadCloseRequested = Events.On('desktop:manual-upload-close-requested', () => { manualCloseRequested.value = true })
	void SetGlobalConfigurationDirty(false)
})

onUnmounted(() => {
	lockRepair()
	window.removeEventListener('focus', handleFocus)
	stopCloseRequested?.()
	stopManualFilesDropped?.()
	stopManualUploadProgress?.()
	stopManualUploadCloseRequested?.()
})
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar" aria-label="Primary navigation">
      <div class="brand-block">
        <img src="/logo.svg" alt="Upit logo" class="brand-logo" width="24" height="24" />
        <div>
          <p class="eyebrow">Upit</p>
          <h1>Desktop</h1>
        </div>
      </div>

      <nav class="nav-list">
        <button
          v-for="area in areas"
          :key="area.id"
          class="nav-item"
          :class="{ active: activeArea === area.id }"
          :aria-current="activeArea === area.id ? 'page' : undefined"
          :disabled="state?.mode !== 'normal' && area.id !== 'file-manager-integration'"
          type="button"
          @click="requestArea(area.id)"
        >
          <span class="nav-label">{{ area.label }}</span>
          <span class="nav-description">{{ area.description }}</span>
        </button>
      </nav>

      <section v-if="manualLoading" class="manual-upload-status" role="status" aria-live="polite">
        <strong>Manual Upload: {{ manualPhaseLabel(manualProgress?.phase ?? 'preparing') }}</strong>
        <span v-if="manualProgress?.total">{{ formatManualBytes(manualProgress.processed) }} of {{ formatManualBytes(manualProgress.total) }}</span>
        <button class="secondary-action" type="button" @click="cancelManualUpload">Cancel</button>
      </section>

      <div class="sidebar-footer">
        <span class="status-dot" :class="state?.mode ?? 'loading'" aria-hidden="true"></span>
        <span v-if="loading">Loading Configuration Set…</span>
        <span v-else-if="error">Desktop unavailable</span>
        <span v-else-if="state?.mode === 'normal'">Configuration Set ready</span>
        <span v-else-if="state?.mode === 'setup'">Setup required</span>
        <span v-else>Repair required</span>
      </div>
    </aside>

    <main class="main-content">
      <header class="topbar">
        <div>
          <p class="eyebrow">Configuration Set</p>
          <h2>{{ activeAreaDetails.label }}</h2>
        </div>
        <span v-if="state?.configurationPath" class="path-chip">{{ state.configurationPath }}</span>
      </header>

      <section v-if="activeArea === 'file-manager-integration'" class="state-card" aria-labelledby="integration-title">
        <h3 id="integration-title">File Manager Integration</h3>
        <button v-if="state?.mode !== 'normal'" class="secondary-action" type="button" @click="activeArea = 'global-configuration'">Return to Configuration Set {{ state?.mode === 'setup' ? 'Setup' : 'Repair' }}</button>
        <p v-if="integrationLoading" role="status">Inspecting installed integration…</p>
        <p v-if="integrationError" role="alert">{{ integrationError }}</p>
        <template v-if="integrationState">
          <strong role="status" aria-live="polite">{{ integrationState.status }}</strong>
          <p>{{ integrationState.guidance }}</p>
          <div class="editor-actions">
            <button v-for="action in integrationState.actions" :key="action" class="secondary-action" type="button" :disabled="integrationLoading" @click="runIntegrationAction(action)">{{ integrationActionLabels[action] }}</button>
          </div>
        </template>
      </section>
      <section v-else-if="loading" class="state-card" aria-live="polite">
        <div class="spinner" aria-hidden="true"></div>
        <h3>Loading your Configuration Set</h3>
        <p>Upit is reading and validating the fixed user configuration location.</p>
      </section>

      <section v-else-if="error" class="state-card state-card-error" role="alert">
        <span class="state-icon" aria-hidden="true">!</span>
        <h3>Desktop could not load</h3>
        <p>{{ error }}</p>
      </section>

      <section v-else-if="state?.mode === 'setup'" class="state-card setup-card">
        <span class="state-icon state-icon-accent" aria-hidden="true">+</span>
        <p class="eyebrow">First run</p>
        <h3>Create your Configuration Set</h3>
        <p>Finish creates the fixed configuration directory and one valid Uploader. Nothing is written while this draft is incomplete.</p>
        <form class="editor-form" :aria-describedby="setupError ? 'setup-error' : undefined" @submit.prevent="finishSetup">
          <label class="field-label" for="setup-default-uploader">Default Uploader name</label>
          <input id="setup-default-uploader" v-model="setupGlobal.defaultUploader" placeholder="Name" />
          <label class="field-label" for="setup-default-shortener">Default Shortener (optional)</label>
          <input id="setup-default-shortener" v-model="setupGlobal.defaultShortener" placeholder="Leave empty until a Shortener is configured" />
          <label class="checkbox-field">
            <input v-model="setupGlobal.copyToClipboard" type="checkbox" />
            <span>Copy Final URL after CLI and Manual Upload (File Manager Upload always copies)</span>
          </label>
          <label class="field-label" for="setup-uploader-name">First Uploader name</label>
          <input id="setup-uploader-name" v-model="setupUploader.name" placeholder="Name" />
          <label class="field-label" for="setup-method">HTTP method</label>
          <input id="setup-method" v-model="setupUploader.request.method" placeholder="POST" />
          <label class="field-label" for="setup-url">Request URL</label>
          <input id="setup-url" v-model="setupUploader.request.url" placeholder="https://upload.example.test" />
          <label class="field-label" for="setup-body">Request Body Mode</label>
          <select id="setup-body" v-model="setupUploader.request.body">
            <option value="" disabled>Select a mode</option>
            <option value="multipart">multipart</option>
            <option value="binary">binary</option>
            <option value="form">form</option>
            <option value="json">json</option>
          </select>
          <label v-if="setupUploader.request.body === 'multipart'" class="field-label" for="setup-file-field">File field</label>
          <input v-if="setupUploader.request.body === 'multipart'" id="setup-file-field" v-model="setupUploader.request.fileField" placeholder="file" />
          <section v-if="setupUploader.request.body === 'multipart' || setupUploader.request.body === 'form'" class="map-editor">
            <div class="map-heading"><h4>Request fields</h4><button class="secondary-action" type="button" @click="addSetupField">Add</button></div>
            <div v-for="(entry, index) in setupUploader.request.fields" :key="`setup-field-${index}`" class="map-row">
              <input v-model="entry.key" aria-label="Field key" placeholder="Field name" />
              <input v-model="entry.value" aria-label="Field value" placeholder="Value or {input}" />
              <button class="icon-action" type="button" @click="removeSetupField(index)">Remove</button>
            </div>
          </section>
          <p v-if="setupUploader.request.body === 'json'" class="muted-copy">JSON data must contain exactly one full-string {input} placeholder.</p>
          <select id="setup-response-type" :value="setupUploader.response.url.type" aria-label="Response URL extractor type" @change="changeSetupExtractor">
            <option value="" disabled>Select an extractor</option>
            <option value="body">body</option>
            <option value="json">json</option>
            <option value="header">header</option>
            <option value="regex">regex</option>
          </select>
          <input v-if="setupUploader.response.url.type === 'json'" v-model="setupUploader.response.url.path" aria-label="Response URL JSONPath" placeholder="JSONPath" />
          <input v-if="setupUploader.response.url.type === 'header'" v-model="setupUploader.response.url.header" aria-label="Response URL header name" placeholder="Response header" />
          <input v-if="setupUploader.response.url.type === 'regex'" v-model="setupUploader.response.url.pattern" aria-label="Response URL RE2 pattern" placeholder="RE2 pattern" />
          <input v-if="setupUploader.response.url.type === 'regex'" v-model="setupUploader.response.url.group" aria-label="Response URL regex group" placeholder="Group (optional)" />
          <textarea v-if="setupUploader.request.body === 'json'" v-model="setupUploader.request.dataJSON" aria-label="JSON request data template" rows="6" placeholder='{"content":"{input}"}'></textarea>
          <p v-if="setupError" id="setup-error" class="inline-error" role="alert">{{ setupError }}</p>
          <button class="primary-action" type="submit" :disabled="!setupReady || setupLoading">Finish Setup</button>
        </form>
      </section>

      <section v-else-if="state?.mode === 'repair'" class="state-card state-card-warning repair-card" role="alert">
        <span class="state-icon" aria-hidden="true">!</span>
        <p class="eyebrow">Repair required</p>
        <h3>Your Configuration Set needs attention</h3>
        <p>{{ state.diagnostic }}</p>
        <label class="field-label" for="repair-kind">Document to repair</label>
        <select id="repair-kind" v-model="repairKind" :disabled="repairLoading" @change="changeRepairKind">
          <option value="config">Global Configuration</option>
          <option value="uploaders">Uploaders</option>
          <option value="shorteners">Shorteners</option>
        </select>
        <p v-if="repairLocked" class="muted-copy">The raw document is locked until you explicitly unlock it. It may contain credentials.</p>
        <button v-if="repairLocked" class="primary-action" type="button" :disabled="repairLoading" @click="unlockRepair">Unlock repair document</button>
        <p v-if="repairError" class="inline-error" role="alert">{{ repairError }}</p>
        <template v-if="!repairLocked">
          <textarea v-if="repairState" v-model="repairState.content" class="repair-textarea" rows="14" spellcheck="false" :disabled="repairLoading"></textarea>
          <div class="editor-actions">
            <button class="primary-action" type="button" :disabled="repairLoading || !repairState" @click="saveRepairDocument">Validate and Save</button>
            <button class="secondary-action" type="button" :disabled="repairLoading" @click="loadRepairDocument">Reload</button>
          </div>
        </template>
      </section>

      <section v-else class="workspace">
        <div class="summary-grid">
          <article class="summary-card">
            <span class="summary-label">Default Uploader</span>
            <strong>{{ state?.defaultUploader }}</strong>
          </article>
          <article class="summary-card">
            <span class="summary-label">Default Shortener</span>
            <strong>{{ state?.defaultShortener || 'None' }}</strong>
          </article>
          <article class="summary-card">
            <span class="summary-label">Clipboard</span>
            <strong>{{ state?.copyToClipboard ? 'Enabled' : 'Disabled' }}</strong>
          </article>
        </div>

        <article v-if="activeArea === 'global-configuration'" class="content-card editor-card">
          <div class="content-card-heading">
            <div>
              <p class="eyebrow">Explicit Save</p>
              <h3>Global Configuration</h3>
            </div>
            <span v-if="globalDirty" class="dirty-badge">Unsaved changes</span>
            <span v-else class="ready-badge">Saved</span>
          </div>
          <p>Choose the defaults used by later uploads. Changes stay in memory until you save them.</p>
          <form class="editor-form" :aria-describedby="globalError ? 'global-config-error' : undefined" @submit.prevent="saveGlobalEditor">
            <label class="field-label" for="default-uploader">Default Uploader</label>
            <select id="default-uploader" v-model="globalEditor.defaultUploader" :disabled="globalLoading">
              <option v-for="name in globalEditor.uploaders" :key="name" :value="name">{{ name }}</option>
            </select>

            <label class="field-label" for="default-shortener">Default Shortener</label>
            <select id="default-shortener" v-model="globalEditor.defaultShortener" :disabled="globalLoading">
              <option value="">None</option>
              <option v-for="name in globalEditor.shorteners" :key="name" :value="name">{{ name }}</option>
            </select>

            <label class="checkbox-field">
              <input v-model="globalEditor.copyToClipboard" type="checkbox" :disabled="globalLoading" />
              <span>Copy Final URL after CLI and Manual Upload (File Manager Upload always copies)</span>
            </label>

            <p v-if="globalError" id="global-config-error" class="inline-error" role="alert">{{ globalError }}</p>
            <p v-if="globalNotice" class="inline-success" role="status">{{ globalNotice }}</p>
            <div class="editor-actions">
              <button class="primary-action" type="submit" :disabled="!globalDirty || globalLoading">Save</button>
              <button class="secondary-action" type="button" :disabled="globalLoading" @click="requestRefresh">Refresh</button>
              <button v-if="globalDirty" class="secondary-action" type="button" :disabled="globalLoading" @click="resolveDirtyAction('discard')">Discard</button>
            </div>
          </form>
        </article>

        <article v-else-if="activeArea === 'uploaders'" class="content-card uploader-editor-card">
          <div class="content-card-heading">
            <div>
              <p class="eyebrow">Structured editor</p>
              <h3>Uploaders</h3>
            </div>
            <span v-if="uploaderDirty" class="dirty-badge">Unsaved changes</span>
            <span v-else class="ready-badge">Saved</span>
          </div>
          <div class="uploader-layout">
            <aside class="uploader-list" aria-label="Uploader definitions">
              <button
                v-for="name in uploaderState?.uploaders ?? []"
                :key="name"
                class="definition-button"
                :class="{ active: uploaderState?.draft.originalName === name }"
                type="button"
                @click="selectUploader(name)"
              >
                <span>{{ name }}</span>
                <small v-if="name === state?.defaultUploader">Default</small>
              </button>
              <button class="secondary-action" type="button" :disabled="uploaderLoading" @click="newUploader">New Uploader</button>
            </aside>

            <form v-if="uploaderState" class="editor-form uploader-form" :aria-describedby="uploaderError ? 'uploader-error' : undefined" @submit.prevent="saveUploaderEditor">
              <label class="field-label" for="uploader-name">Name</label>
              <input id="uploader-name" v-model="uploaderState.draft.name" :disabled="Boolean(uploaderState.draft.originalName) || uploaderLoading" />

              <label class="field-label" for="uploader-method">HTTP method</label>
              <input id="uploader-method" v-model="uploaderState.draft.request.method" :disabled="uploaderLoading" />

              <label class="field-label" for="uploader-url">Request URL</label>
              <input id="uploader-url" v-model="uploaderState.draft.request.url" :disabled="uploaderLoading" />

              <label class="field-label" for="uploader-body">Request Body Mode</label>
              <select id="uploader-body" :value="uploaderState.draft.request.body" :disabled="uploaderLoading" @change="changeUploaderBody">
                <option value="multipart">multipart</option>
                <option value="binary">binary</option>
                <option value="form">form</option>
                <option value="json">json</option>
              </select>

              <template v-if="uploaderState.draft.request.body === 'multipart'">
                <label class="field-label" for="uploader-file-field">File field</label>
                <input id="uploader-file-field" v-model="uploaderState.draft.request.fileField" :disabled="uploaderLoading" />
              </template>

              <section v-for="map in uploaderMapKinds" v-show="map.key !== 'fields' || uploaderState.draft.request.body === 'multipart' || uploaderState.draft.request.body === 'form'" :key="map.key" class="map-editor">
                <div class="map-heading">
                  <h4>{{ map.label }}</h4>
                  <button class="secondary-action" type="button" @click="addMapEntry(map.key)">Add</button>
                </div>
                <div v-for="(entry, index) in uploaderState.draft.request[map.key] ?? []" :key="`${map.key}-${index}`" class="map-row">
                  <input v-model="entry.key" :aria-label="`${map.label} key ${index + 1}`" placeholder="Key" :disabled="uploaderLoading" />
                  <input
                    v-model="entry.value"
                    :type="entry.sensitive && !revealedSecrets[secretKey(map.key, index)] ? 'password' : 'text'"
                    :aria-label="`${map.label} value ${index + 1}`"
                    placeholder="Value"
                    :disabled="uploaderLoading"
                  />
                  <button v-if="entry.sensitive" class="icon-action" type="button" @click="toggleSecret(map.key, index)">{{ revealedSecrets[secretKey(map.key, index)] ? 'Mask' : 'Reveal' }}</button>
                  <button class="icon-action" type="button" @click="removeMapEntry(map.key, index)">Remove</button>
                </div>
              </section>

              <section v-if="uploaderState.draft.request.body === 'json'" class="map-editor">
                <label class="field-label" for="uploader-data">JSON request data</label>
                <textarea id="uploader-data" v-model="uploaderState.draft.request.dataJSON" rows="8" spellcheck="false" :disabled="uploaderLoading"></textarea>
              </section>

              <section class="map-editor">
                <h4>Response URL extractor</h4>
                <select :value="uploaderState.draft.response.url.type" aria-label="Response URL extractor type" :disabled="uploaderLoading" @change="changeUploaderExtractor">
                  <option value="json">json</option>
                  <option value="header">header</option>
                  <option value="regex">regex</option>
                  <option value="body">body</option>
                </select>
                <input v-if="uploaderState.draft.response.url.type === 'json'" v-model="uploaderState.draft.response.url.path" aria-label="Response URL JSONPath" placeholder="JSONPath" :disabled="uploaderLoading" />
                <input v-if="uploaderState.draft.response.url.type === 'header'" v-model="uploaderState.draft.response.url.header" aria-label="Response URL header name" placeholder="Response header" :disabled="uploaderLoading" />
                <input v-if="uploaderState.draft.response.url.type === 'regex'" v-model="uploaderState.draft.response.url.pattern" aria-label="Response URL RE2 pattern" placeholder="RE2 pattern" :disabled="uploaderLoading" />
                <input v-if="uploaderState.draft.response.url.type === 'regex'" v-model="uploaderState.draft.response.url.group" aria-label="Response URL regex group" placeholder="Group (optional)" :disabled="uploaderLoading" />
              </section>

              <section class="map-editor">
                <label class="checkbox-field">
                  <input type="checkbox" :checked="Boolean(uploaderState.draft.response.error)" :disabled="uploaderLoading" @change="onErrorToggle" />
                  <span>Configure response error extractor</span>
                </label>
                <template v-if="uploaderState.draft.response.error">
                  <select :value="uploaderState.draft.response.error.type" aria-label="Response error extractor type" :disabled="uploaderLoading" @change="changeUploaderErrorExtractor">
                    <option value="json">json</option>
                    <option value="header">header</option>
                    <option value="regex">regex</option>
                    <option value="body">body</option>
                  </select>
                  <input v-if="uploaderState.draft.response.error.type === 'json'" v-model="uploaderState.draft.response.error.path" aria-label="Response error JSONPath" placeholder="JSONPath" :disabled="uploaderLoading" />
                  <input v-if="uploaderState.draft.response.error.type === 'header'" v-model="uploaderState.draft.response.error.header" aria-label="Response error header name" placeholder="Response header" :disabled="uploaderLoading" />
                  <input v-if="uploaderState.draft.response.error.type === 'regex'" v-model="uploaderState.draft.response.error.pattern" aria-label="Response error RE2 pattern" placeholder="RE2 pattern" :disabled="uploaderLoading" />
                  <input v-if="uploaderState.draft.response.error.type === 'regex'" v-model="uploaderState.draft.response.error.group" aria-label="Response error regex group" placeholder="Group (optional)" :disabled="uploaderLoading" />
                </template>
              </section>

              <p v-if="uploaderError" id="uploader-error" class="inline-error" role="alert">{{ uploaderError }}</p>
              <div class="editor-actions">
                <button class="primary-action" type="submit" :disabled="!uploaderDirty || uploaderLoading">Save Uploader</button>
                <button class="secondary-action" type="button" :disabled="uploaderLoading" @click="requestRefresh">Refresh</button>
                <button class="secondary-action" type="button" :disabled="uploaderLoading || uploaderDirty || !uploaderState.draft.originalName" @click="renameUploader">Rename</button>
                <button class="secondary-action" type="button" :disabled="uploaderLoading || uploaderDirty || !uploaderState.draft.originalName" @click="deleteUploader">Delete</button>
              </div>
            </form>
          </div>
        </article>
        <article v-else-if="activeArea === 'shorteners'" class="content-card uploader-editor-card">
          <div class="content-card-heading">
            <div>
              <p class="eyebrow">Structured editor</p>
              <h3>Shorteners</h3>
            </div>
            <span v-if="shortenerDirty" class="dirty-badge">Unsaved changes</span>
            <span v-else class="ready-badge">Saved</span>
          </div>
          <div class="uploader-layout">
            <aside class="uploader-list" aria-label="Shortener definitions">
              <button
                v-for="name in shortenerState?.shorteners ?? []"
                :key="name"
                class="definition-button"
                :class="{ active: shortenerState?.draft.originalName === name }"
                type="button"
                @click="selectShortener(name)"
              >
                <span>{{ name }}</span>
                <small v-if="name === state?.defaultShortener">Default</small>
              </button>
              <button class="secondary-action" type="button" :disabled="shortenerLoading" @click="newShortener">New Shortener</button>
            </aside>

            <form v-if="shortenerState" class="editor-form uploader-form" :aria-describedby="shortenerError ? 'shortener-error' : undefined" @submit.prevent="saveShortenerEditor">
              <label class="field-label" for="shortener-name">Name</label>
              <input id="shortener-name" v-model="shortenerState.draft.name" :disabled="Boolean(shortenerState.draft.originalName) || shortenerLoading" />
              <label class="field-label" for="shortener-method">HTTP method</label>
              <input id="shortener-method" v-model="shortenerState.draft.request.method" :disabled="shortenerLoading" />
              <label class="field-label" for="shortener-url">Request URL</label>
              <input id="shortener-url" v-model="shortenerState.draft.request.url" :disabled="shortenerLoading" />

              <section v-for="map in shortenerMapKinds" :key="map.key" class="map-editor">
                <div class="map-heading">
                  <h4>{{ map.label }}</h4>
                  <button class="secondary-action" type="button" @click="addShortenerMapEntry(map.key)">Add</button>
                </div>
                <div v-for="(entry, index) in shortenerState.draft.request[map.key] ?? []" :key="`${map.key}-${index}`" class="map-row">
                  <input v-model="entry.key" :aria-label="`${map.label} key ${index + 1}`" placeholder="Key" :disabled="shortenerLoading" />
                  <input v-model="entry.value" :type="entry.sensitive && !revealedSecrets[secretKey(`shortener-${map.key}`, index)] ? 'password' : 'text'" :aria-label="`${map.label} value ${index + 1}`" placeholder="Value" :disabled="shortenerLoading" />
                  <button v-if="entry.sensitive" class="icon-action" type="button" @click="toggleSecret(`shortener-${map.key}`, index)">{{ revealedSecrets[secretKey(`shortener-${map.key}`, index)] ? 'Mask' : 'Reveal' }}</button>
                  <button class="icon-action" type="button" @click="removeShortenerMapEntry(map.key, index)">Remove</button>
                </div>
              </section>

              <label class="field-label" for="shortener-data">JSON request data</label>
              <textarea id="shortener-data" v-model="shortenerState.draft.request.dataJSON" rows="8" spellcheck="false" :disabled="shortenerLoading"></textarea>
              <section class="map-editor">
                <h4>Response URL extractor (JSON only)</h4>
                <input v-model="shortenerState.draft.response.url.path" aria-label="Shortener response URL JSONPath" placeholder="JSONPath" :disabled="shortenerLoading" />
                <label class="checkbox-field">
                  <input type="checkbox" :checked="Boolean(shortenerState.draft.response.error)" :disabled="shortenerLoading" @change="onShortenerErrorToggle" />
                  <span>Configure provider error extractor</span>
                </label>
                <input v-if="shortenerState.draft.response.error" v-model="shortenerState.draft.response.error.path" aria-label="Shortener provider error JSONPath" placeholder="Error JSONPath" :disabled="shortenerLoading" />
              </section>

              <p v-if="shortenerError" id="shortener-error" class="inline-error" role="alert">{{ shortenerError }}</p>
              <div class="editor-actions">
                <button class="primary-action" type="submit" :disabled="!shortenerDirty || shortenerLoading">Save Shortener</button>
                <button class="secondary-action" type="button" :disabled="shortenerLoading" @click="requestRefresh">Refresh</button>
                <button class="secondary-action" type="button" :disabled="shortenerLoading || shortenerDirty || !shortenerState.draft.originalName" @click="renameShortener">Rename</button>
                <button class="secondary-action" type="button" :disabled="shortenerLoading || shortenerDirty || !shortenerState.draft.originalName" @click="deleteShortener">Delete</button>
              </div>
            </form>
          </div>
        </article>
        <article v-else class="content-card editor-card manual-upload-card">
          <div class="content-card-heading">
            <div>
              <p class="eyebrow">One file</p>
              <h3>Manual Upload</h3>
            </div>
            <span v-if="manualLoading" class="dirty-badge">Uploading</span>
            <span v-else class="ready-badge">Ready</span>
          </div>
          <p>Choose one regular file or drop it below. Your Configuration Set stays unchanged.</p>
          <form class="editor-form manual-upload-form" :aria-describedby="manualError ? 'manual-upload-error' : undefined" @submit.prevent="startManualUpload">
            <section id="manual-upload-drop" class="manual-upload-drop" data-file-drop-target>
              <strong>{{ manualFile ? manualFile.name : 'Drop one file here' }}</strong>
              <span v-if="manualFile">{{ manualFile.path }} · {{ manualFile.size.toLocaleString() }} bytes</span>
              <span v-else>Directories and multiple-file drops are not supported.</span>
              <button class="secondary-action" type="button" :disabled="manualLoading" @click="chooseManualUploadFile">Choose file</button>
            </section>

            <label class="field-label" for="manual-uploader">Uploader</label>
            <select id="manual-uploader" v-model="manualUploaderChoice" :disabled="manualLoading">
              <option value="default">Use Global Configuration ({{ state?.defaultUploader }})</option>
              <option v-for="name in state?.uploaders ?? []" :key="name" :value="`named:${name}`">{{ name }}</option>
            </select>

            <label class="field-label" for="manual-shortener">Shortener</label>
            <select id="manual-shortener" v-model="manualShortenerChoice" :disabled="manualLoading">
              <option value="default">Use Global Configuration ({{ state?.defaultShortener || 'None' }})</option>
              <option value="none">None</option>
              <option v-for="name in state?.shorteners ?? []" :key="name" :value="`named:${name}`">{{ name }}</option>
            </select>

            <label class="field-label" for="manual-clipboard">Clipboard</label>
            <select id="manual-clipboard" v-model="manualClipboard" :disabled="manualLoading">
              <option value="default">Use Global Configuration ({{ state?.copyToClipboard ? 'Enabled' : 'Disabled' }})</option>
              <option value="enabled">Copy Final URL</option>
              <option value="disabled">Do not copy</option>
            </select>

            <label class="field-label" for="manual-timeout">Timeout (optional)</label>
            <input id="manual-timeout" v-model="manualTimeout" placeholder="30s or 10m" :disabled="manualLoading" />

            <p v-if="anyDirty" class="muted-copy" role="status">Save or discard Configuration Set edits before uploading.</p>
            <p v-if="manualError" id="manual-upload-error" class="inline-error" role="alert">{{ manualError }}</p>

            <template v-if="manualResult?.success">
              <section class="manual-upload-result inline-success" role="status">
                <strong>Upload complete</strong>
                <a :href="manualResult.finalURL" target="_blank" rel="noreferrer">{{ manualResult.finalURL }}</a>
                <span v-if="manualResult.originalURL !== manualResult.finalURL">Original URL: {{ manualResult.originalURL }}</span>
                <span v-for="warning in manualResult.warnings ?? []" :key="warning">Warning: {{ warning }}</span>
                <button class="secondary-action" type="button" @click="copyManualUploadFinalURL">Copy Final URL</button>
              </section>
            </template>
            <section v-else-if="manualResult?.failure" class="manual-upload-result inline-error" role="alert">
              <strong>{{ manualResult.failure.canceled ? 'Upload canceled' : 'Upload failed' }}</strong>
              <span>Stage: {{ manualResult.failure.stage }}</span>
              <span v-if="manualResult.failure.statusCode">HTTP: {{ manualResult.failure.statusCode }}</span>
              <span>{{ manualResult.failure.message }}</span>
              <button class="secondary-action" type="button" :disabled="manualLoading" @click="startManualUpload">Retry</button>
            </section>
            <p v-if="manualNotice" class="inline-success" role="status">{{ manualNotice }}</p>

            <div class="editor-actions">
              <button class="primary-action" type="submit" :disabled="!manualReady">{{ manualLoading ? 'Uploading…' : 'Upload file' }}</button>
              <button v-if="manualLoading" class="secondary-action" type="button" @click="cancelManualUpload">Cancel</button>
            </div>
          </form>
        </article>
      </section>
      <div v-if="dirtyAction" class="modal-backdrop" role="presentation" @keydown="onDirtyDialogKeydown">
        <section ref="dirtyDialogRef" class="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="dirty-dialog-title">
          <p class="eyebrow">Unsaved changes</p>
          <h3 id="dirty-dialog-title">Save {{ dirtyEditorLabel }} changes?</h3>
          <p>Your current edits are not saved. Save them before {{ dirtyAction === 'navigate' ? 'leaving this area' : 'refreshing' }}?</p>
          <div class="editor-actions">
            <button class="primary-action" type="button" @click="resolveDirtyAction('save')">Save</button>
            <button class="secondary-action" type="button" @click="resolveDirtyAction('discard')">Discard</button>
            <button ref="dirtyCancelRef" class="secondary-action" type="button" @click="resolveDirtyAction('cancel')">Cancel</button>
          </div>
        </section>
      </div>
      <div v-if="manualCloseRequested" class="modal-backdrop" role="presentation" @keydown="onManualCloseDialogKeydown">
        <section ref="manualCloseDialogRef" class="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="manual-close-dialog-title">
          <p class="eyebrow">Manual Upload active</p>
          <h3 id="manual-close-dialog-title">Cancel upload and close?</h3>
          <p>Closing cancels the active upload and waits for its resources to be released.</p>
          <div class="editor-actions">
            <button class="primary-action" type="button" @click="resolveManualUploadClose('confirm')">Cancel upload and close</button>
            <button ref="manualCloseCancelRef" class="secondary-action" type="button" @click="resolveManualUploadClose('cancel')">Keep uploading</button>
          </div>
        </section>
      </div>
      <div v-if="closeRequested" class="modal-backdrop" role="presentation" @keydown="onCloseDialogKeydown">
        <section ref="closeDialogRef" class="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="close-dialog-title">
          <p class="eyebrow">Close Upit Desktop</p>
          <h3 id="close-dialog-title">Save {{ dirtyEditorLabel }} changes?</h3>
          <p>Your current edits are not saved. Choose Save, Discard, or Cancel before closing.</p>
          <div class="editor-actions">
            <button class="primary-action" type="button" @click="resolveClose('save')">Save</button>
            <button class="secondary-action" type="button" @click="resolveClose('discard')">Discard</button>
            <button ref="closeCancelRef" class="secondary-action" type="button" @click="resolveClose('cancel')">Cancel</button>
          </div>
        </section>
      </div>
      <div v-if="renameDialog.open" class="modal-backdrop" role="presentation" @keydown="onRenameDialogKeydown">
        <section ref="renameDialogRef" class="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="rename-dialog-title">
          <p class="eyebrow">Rename {{ renameDialog.kind === 'uploader' ? 'Uploader' : 'Shortener' }}</p>
          <h3 id="rename-dialog-title">Rename "{{ renameDialog.currentName }}"</h3>
          <form class="editor-form" :aria-describedby="renameDialog.error ? 'rename-dialog-error' : undefined" @submit.prevent="submitRename">
            <label class="field-label" for="rename-input">New name</label>
            <input
              id="rename-input"
              ref="renameInputRef"
              v-model="renameDialog.newName"
              :disabled="renameDialog.loading"
              required
            />
            <p v-if="renameDialog.error" id="rename-dialog-error" class="inline-error" role="alert">{{ renameDialog.error }}</p>
            <div class="editor-actions">
              <button class="primary-action" type="submit" :disabled="renameDialog.loading || !renameDialog.newName.trim()">
                {{ renameDialog.loading ? 'Renaming…' : 'Rename' }}
              </button>
              <button class="secondary-action" type="button" :disabled="renameDialog.loading" @click="closeRenameDialog">
                Cancel
              </button>
            </div>
          </form>
        </section>
      </div>
      <div v-if="deleteDialog.open" class="modal-backdrop" role="presentation" @keydown="onDeleteDialogKeydown">
        <section ref="deleteDialogRef" class="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="delete-dialog-title" :aria-describedby="deleteDialog.error ? 'delete-dialog-error' : undefined">
          <p class="eyebrow">Confirm deletion</p>
          <h3 id="delete-dialog-title">Delete {{ deleteDialog.kind === 'uploader' ? 'Uploader' : 'Shortener' }} "{{ deleteDialog.name }}"?</h3>
          <p>This action removes the definition. Unreferenced definitions are permanently deleted upon confirmation.</p>
          <p v-if="deleteDialog.error" id="delete-dialog-error" class="inline-error" role="alert">{{ deleteDialog.error }}</p>
          <div class="editor-actions">
            <button class="primary-action" type="button" :disabled="deleteDialog.loading" @click="submitDelete">
              {{ deleteDialog.loading ? 'Deleting…' : 'Delete' }}
            </button>
            <button ref="deleteCancelRef" class="secondary-action" type="button" :disabled="deleteDialog.loading" @click="closeDeleteDialog">
              Cancel
            </button>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>
