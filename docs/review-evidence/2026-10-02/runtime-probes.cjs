const fs = require('fs');
const vm = require('vm');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const ts = require(path.join(root, 'frontend/node_modules/typescript'));
function mockCreate(init) {
  let state;
  const get = () => state;
  const set = next => state = Object.assign({}, state, typeof next === 'function' ? next(state) : next);
  state = init(set, get);
  return {getState:get, setState:set};
}
function loadTS(file, mocks) {
  const output = ts.transpileModule(fs.readFileSync(path.join(root,file),'utf8'), {compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;
  const context = {exports:{},require:name => {if (name in mocks) return mocks[name];throw new Error(name)},console,setTimeout,clearTimeout,Map,Date,Promise};
  vm.runInNewContext(output,context);
  return context.exports;
}
const logger = {Logger:{error:()=>{}}};
(async () => {
  const {useAppStore:app}=loadTS('frontend/src/stores/appStore.ts', {'zustand':{create:mockCreate}});
  app.getState().setTextValue('old pending query');
  app.getState().clearValue();
  await new Promise(r=>setTimeout(r,80));
  console.log('debounce-after-clear', JSON.stringify({value:app.getState().value,displayValue:app.getState().displayValue}));
  app.getState().setClipboardContent({contentType:'image',imageBase64:'fake-image'});
  app.getState().setTextValue('json');
  await new Promise(r=>setTimeout(r,80));
  console.log('typed-after-image', JSON.stringify({value:app.getState().value,contentType:app.getState().clipboardContentType,image:app.getState().imageBase64}));
  let resolveWrite;
  const {usePluginStore:p}=loadTS('frontend/src/stores/pluginStore.ts', {'zustand':{create:mockCreate},'@/api/plugin':{getPlugins:async()=>[{packageId:'watools.plugin.probe',enabled:false,entry:[],usedCount:0}],togglePlugin:async()=>{},uninstallPlugin:async()=>{},updatePluginUsage:()=>new Promise(r=>resolveWrite=r)},'@/lib/logger':logger});
  await new Promise(r=>setImmediate(r));
  await p.getState().togglePlugin('watools.plugin.probe',true);
  console.log('enabled-without-load',JSON.stringify(p.getState().plugins[0]));
  await p.getState().updatePluginUsage('watools.plugin.probe');
  const pending=p.getState().flushBufferUpdates();
  await p.getState().updatePluginUsage('watools.plugin.probe');
  resolveWrite(); await pending;
  console.log('usage-update-during-flush',JSON.stringify({usedCount:p.getState().plugins[0].usedCount,bufferSize:p.getState().updateBuffer.size}));
  // Cancel the pending flush timer without another write.
  await p.getState().flushBufferUpdates();
  const html=fs.readFileSync(path.join(root,'plugins/official/watools.plugin.json/plugin/index.html'),'utf8');
  const jsonContext={URL,URLSearchParams,console};
  vm.createContext(jsonContext);
  vm.runInContext(html.slice(html.indexOf('    const formatPathSegment'),html.indexOf('    const findDeepestNodeAtOffset'))+'\nthis.parseProbe = text => new JsonTreeParser(text).parse().value;',jsonContext);
  for(const input of ['{"a":1,','[1,','{"x":"raw\nnewline"}','{"__proto__":{"hidden":true},"a":1}']) {
    try{console.log('json-parser',JSON.stringify({input,result:jsonContext.parseProbe(input)}))}catch(e){console.log('json-parser rejected',JSON.stringify(input),e.message)}
  }
  vm.runInContext(html.slice(html.indexOf('    const smartDecode'),html.indexOf('    const appendQueryPair'))+'\nthis.queryProbe = parseQueryCandidate;',jsonContext);
  console.log('query-prototype-key', JSON.stringify(jsonContext.queryProbe('constructor=abc&toString=def')));
  const translation=fs.readFileSync(path.join(root,'plugins/official/watools.plugin.translate/plugin/index.html'),'utf8');
  const elements = new Map();
  const elem=id=>{if(!elements.has(id))elements.set(id,{value:'',textContent:'',className:'',classList:{toggle(){},add(){},remove(){}},focus(){},select(){}});return elements.get(id)};
  let resolveRequest; let calls=0;
  const tContext={document:{getElementById:elem},window:{watools:{HttpProxy:()=>{calls++; return new Promise(r=>resolveRequest=r)}}},console,setTimeout,clearTimeout,URLSearchParams};
  vm.createContext(tContext);
  const from=translation.indexOf('    const apiKeyStorageKey');
  const to=translation.indexOf('    const copyTranslation');
  vm.runInContext(translation.slice(from,to)+'\napiKey="probe-key"; this.translateProbe=translateText;',tContext);
  elem('source-text').value='old';
  const first=tContext.translateProbe();
  elem('source-text').value='new';
  await tContext.translateProbe();
  resolveRequest({status_code:200,body:JSON.stringify({translations:[{text:'translation of old'}]})});
  await first;
  console.log('translation-stale-response', JSON.stringify({calls,currentInput:elem('source-text').value,output:elem('target-text').value}));
})();
