#!/usr/bin/env python3
"""Render sealed S2-R6 results. This file makes no scientific decisions."""
import argparse,json,re,shutil
from pathlib import Path
import numpy as np
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
ROOT=Path(__file__).resolve().parents[1]
SEEDS=[37,83,151,211,307,419]
ARMS=['A','A0','B','B0','BI','BI0','C','F']
def read(p):return json.loads(p.read_text())
def pct(x):return f'{100*x:.3f}%'
def ci(v):return f'[{pct(v[0])}, {pct(v[1])}]'
def table(headers,rows):return '| '+' | '.join(headers)+' |\n|'+ '|'.join(['---']*len(headers))+'|\n'+'\n'.join('| '+' | '.join(map(str,row))+' |' for row in rows)+'\n'
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args();w=a.work;out=a.output;out.mkdir(parents=True,exist_ok=True)
 r=read(w/'SCIENTIFIC_RESULTS.json');lock=read(w/'CONFIRMATORY_LOCK.json');train=read(w/'TRAINING_LOCK.json');source=read(w/'SOURCE_VERIFICATION.json');ret=read(w/'RETENTION_LOCK.json')
 u=r['utility']['results'];h=r['history']['persistent'];c=h['arms']['C']['pooled'];resource=r['resources'];old=r['retention']['C'];pair=h['contrasts']['C-vs-B0'];iid=r['history']['independent']['contrasts']['BI-vs-BI0']
 chunks=[]
 def section(n,title,text):chunks.append(f'## {n}. {title}\n\n{text.strip()}\n')
 chunks.append(f'''# S2-R6 — Native Neural Utility, Retention & Adaptive Plasticity Learning

**Tanggal:** 18 November 2026 (Asia/Jakarta). **Verdict: {r['verdict']}.**
Repository: `mpsprintit-spec/horizon-artificial-brain`; branch `research/s2-r6-native-utility-retention-20261118`.
Main baseline: `11c6e0a78547d68a0be8f78b3c801e912722608d`.
Basis kode R5: `099ae285737cdc62fc121791ba4e6f4e463a1cb3`.
Source training: `{train['SourceCommit']}`; source/analysis confirmatory: `{lock['SourceCommit']}`.
Main dan Brain produksi tidak diubah. Tidak ada merge PR sebelumnya atau arm penelitian yang dipromosikan.
''')
 section(1,'Executive scientific verdict',f'''Tujuan yang diuji adalah kemampuan N mempelajari manfaat suatu perubahan neural dari intervensi aktual. Implementasi menyediakan pengalaman kausal dan readout utility pada32 fitur N yang sama. Seluruh enam seed menghasilkan48 model final penelitian; semua merupakan salinan dari sumber R2 yang sah.

**H1: {r['H1']}; H2: {r['H2']}; H3: {r['H3']}; H4: {r['H4']}.** H1 tidak lulus development independen sehingga tahap interference/adaptive plasticity/consolidation tetap ditahan. Hasil final tidak dipakai untuk membalik keputusan prasyarat tersebut. Tidak ada overall PASS.

Pada final unseen pathways, native utility RMSE {u['unseen']['C']['rmse']:.8f} versus baseline terbaik {min(u['unseen']['C']['baseline_rmse'].values()):.8f}, manfaat {pct(u['unseen']['C']['benefit'])}, CI95 {ci(u['unseen']['C']['CI95'])}. Perbaikan Check selama training bukan bukti generalisasi.

Manfaat history C/B terhadap matched B0 adalah {pct(pair['relative_RMSE_improvement'])}, CI95 {ci(pair['CI95'])}; calibration mean-ratio {c['mean_ratio']:.5f}. Retensi seluruh kontrak lulus {sum(v['all_retention_pass'] for v in old['per_seed'])}/6 untuk C.

Keputusan: **perlu perbaikan**. Utility belum boleh menjadi pengendali plasticity. Hasil safeguard eksternal dipisahkan dari kemampuan yang dipelajari.''')
 section(2,'Brain source verification',f'''Arsip R5 lengkap:1.127.417.584 byte, SHA256 `ee6cadaae0fa122beb40da1908c776262385f8c7e758d814e2bc4e411d9e2531`;3.272/3.272 payload terautentikasi. Arsip R2 di dalamnya mempunyai SHA256 `e51b61bf41107f4f74e7d72d6360b22cd4671d0f6ce84e867212090290e0aa46`;443/443 payload diverifikasi.

Checkpoint dan journal R2 byte-identik dengan `r5/00-source.json` dan `r5/source-events.jsonl` terautentikasi. Sequence80735; native checksum `{source['native_checksum']}`;274 unit,6.408 sinaps,32 fitur dan10.752 native policy updates. Checkpoint SHA256 `{source['checkpoint_sha256']}`; journal SHA256 `{source['journal_sha256']}`. Native historical suffix80681→80735 exact; activation/learning/DNF memakai pemilik Brain yang sama.

Lineage: Canonical A–E1 → S1 → S2 → S2-R2 → S2-R6. Tidak ada checkpoint final R3/R4/R5 yang dipilih. Kemampuan history dibentuk ulang memakai prosedur D32 R5 pada training baru. C/F meneruskan common prefix B per seed; tidak ada reset di antara blok utility.''')
 section(3,'Architecture audit','''Audit lengkap tersedia dalam `ARCHITECTURE_AUDIT.md`. Aktivitas rekuren, komposisi numerik, signed parameters, eligibility, policy temporal credit dan paired factual contrast sudah ada. Jalur pengalaman konsekuensi numerical-path intervention ke query utility belum tersedia; adapter riset menyediakan pengukuran tersebut.

Generic shared-basis reuse membentuk query numerik baru pada fitur dan signed substrate N yang sama. V1 memakai fitter lama; V2 memakai solver berbatas R5 pada zero-initialized query. Versi dibedakan agar replay prototype lama tetap exact. Seluruh hasil belajar berada pada ordinary canonical synapses, bukan objek estimator eksternal. Jumlah fitur tetap32. Query baru dapat menambah empat query units; itu tidak menambah feature cap atau substrate.

Tidak ada Utility Engine, utility threshold, ranking neuron hardcoded, candidate-selection policy baru, permanently frozen general synapses atau controller plasticity. Fixed basis adalah ruang pencarian lesson ini, bukan bukti consolidation. General policy tetap10.752 updates.''')
 section(4,'Native utility experience','''Setiap seed mempunyai512 pengalaman training utility:256 Fit dan256 Check, empat blok64/64. Dua intervensi dari satu operasi asal tetap pada split yang sama. Semua label berasal dari squared-loss difference aktual setelah native numerical counterfactual forwards; actual S2 error berasal dari operasi propagate/stop yang benar-benar dijalankan.

Utility = `(loss_before − loss_after)/4`; faktor empat hanya normalisasi rentang. Tidak ada pemilihan contoh hanya karena bermanfaat. Single/pair pathways menerima signed changes −0.08/−0.02/0/+0.02/+0.08 dan ablation. Semua source/intermediate checkpoints, actual outcomes, source IDs, eligibility dan deltas dipertahankan.''')
 chunks[-1]+='\nUntuk ablation, gain positif berarti menghapus pathway membantu, sehingga pathway itu mengganggu pada konteks tersebut. Gain negatif berarti penghapusan merusak prediksi. Ini utility perubahan, bukan ranking intrinsik neuron.\n'
 section(5,'Causal intervention design','''Urutan: pre-intervention descriptor → utility forecast native → ledger di-sync → paired counterfactual → actual loss/utility. Input N tidak memuat target, outcome aktual, ID neuron, ID kandidat atau keputusan gate. Descriptor memuat bounded preactivation/metric/weight/delta/eligibility/dependency summaries dan konteks terbatas.

Paired forwards memakai initial signed parameters dan input identik; hanya pathway yang dideklarasikan berubah. Workspace dibuang sesudah pengukuran. Tidak ada parameter berisiko dipasang pada sumber kanonik. Ini uji computational numerical-readout pathways, bukan ablation seluruh aktivitas rekuren atau seluruh fungsi sebuah neuron. Pasangan pathway tidak membuktikan causal decomposition seluruh jaringan.''')
 rows=[]
 for name,v in u.items():
  z=v['C'];rows.append([name,z['N'],f"{z['rmse']:.8f}",z['best_baseline'],f"{min(z['baseline_rmse'].values()):.8f}",pct(z['benefit']),ci(z['CI95']),f"{z['macro_F1']:.3f}"])
 section(6,'Utility learning results',table(['Holdout','N','Native RMSE','Best baseline','Baseline RMSE','Benefit','CI95','Macro-F1'],rows)+'''
Gate memerlukan≥10% manfaat, CI95 lower>0, macro-F1≥0.40, ketiga kelas≥10 contoh, generalisasi seen/unseen/shifted dan shuffled control tidak memberi manfaat sebanding. Baseline constant, zero, activation-magnitude OLS dan weight-magnitude OLS dilatih hanya pada Fit. Final bersama memakai bootstrap seed dan stream yang dipertahankan lintas model,10.000 replika.

Neural pathway ordinals mod3=0 tidak pernah diintervensi pada Fit/Check:11 unseen pathways versus21 seen. ID-nya tidak masuk codec. Per-seed/stream/class counts ada dalam `SCIENTIFIC_RESULTS.json` dan seluruh prediksi mentah.''')
 perseed=[]
 for seed in SEEDS:
  vals=[next(v for v in u[label]['C']['per_seed'] if v['seed']==seed)['rmse'] for label in ['seen','unseen','shifted']]
  perseed.append([seed]+[f'{v:.8f}' for v in vals])
 chunks[-1]+='\n'+table(['Seed','Seen utility RMSE','Unseen utility RMSE','Shifted utility RMSE'],perseed)+'\nSeed211 mendominasi kegagalan generalisasi; tidak ada seed replacement, checkpoint retry setelah final atau penghapusan outlier. Hasil ini menunjukkan bahwa admission improvement pada Check tidak menjamin forecast utility reliable di konteks baru.\n'
 section(7,'Neural interference learning','''**NOT RUN.** H1 development gagal sebelum final acquisition. Karena itu tidak ada training/prediksi H2 yang diklaim. Tidak ada label interferensi sintetis dari asumsi bahwa jenis perubahan tertentu selalu merusak. Arm D/E belum dijalankan. Kegagalan prasyarat disimpan sebagai hasil penelitian, bukan disembunyikan sebagai keberhasilan implisit.''')
 section(8,'Retention prediction','''**Belum terbukti.** C memprediksi manfaat intervensi numerical-path lokal; belum memprediksi perubahan kemampuan lama oleh candidate native learner yang belum dijalankan. Net-benefit multi-kemampuan, delayed effects, novel update magnitudes dan ID-independent interference transfer tidak diuji karena tahap prasyarat belum lulus.''')
 section(9,'Native adaptive plasticity','''**NOT RUN.** Utility C tidak memengaruhi candidate search, ranking, gate, solver atau installed update. Karena itu tidak ada attribution improvement terhadap native policy adaptation. Arm G nonlearned candidate-selection control juga tidak dijalankan: tahap selection yang hendak dibandingkan belum dibuka.

Pada stage yang berjalan, solver membentuk candidate, native admission memeriksa Check dan retention banks, lalu writer memasang candidate yang lolos. Semua candidate/trials/rejections tercatat. Tidak ada utility selector yang diam-diam ditambahkan. C/F mempunyai S2 predictions persis sama dengan common prefix B pada512 training contexts, diverifikasi sebelum final.''')
 section(10,'Knowledge consolidation','''**NOT PROVEN.** Empat utility lesson meneruskan Brain yang sama, tetapi bukan rangkaian empat kemampuan berbeda A→B→C→D. Tidak ada bukti bahwa N telah belajar memilih pengetahuan valid untuk dipertahankan atau salah untuk diperbaiki. Witness gates/shared-dependency guards tetap safeguard eksternal.''')
 serial=[]
 for arm in ['A','B','C','F']:
  blocks=r['continual'][arm];counts=[sum(v['changed_signed_parameter_pairs'] for v in s['blocks']) for s in blocks]
  serial.append([arm,4,resource[arm]['commits'],24,f'{min(counts)}–{max(counts)}'])
 section(11,'Continual learning sequence',table(['Arm','Serial blocks/seed','Committed','Opportunities','Changed signed parameter-pairs/seed'],serial)+'''
Semua intermediate checkpoints tersedia. Parameter-pairs menjumlahkan excitatory/inhibitory canonical values pada pasangan source/target; bukan menghitung dua sign branches sebagai satu nilai yang saling menimpa. Jumlah ini adalah perubahan struktural terukur, bukan skor consolidation. Forward/backward transfer antar empat kemampuan dan corrective learning belum diuji; matriks tersebut tidak diisi dengan angka rekaan.''')
 hr=[]
 for condition in ['persistent','independent','shifted']:
  q=r['history'][condition];b=q['contrasts']['B-vs-B0'];aa=q['contrasts']['A-vs-A0'];hr.append([condition,f"{q['arms']['A']['pooled']['RMSE']:.8f}",f"{q['arms']['B']['pooled']['RMSE']:.8f}",pct(aa['relative_RMSE_improvement']),pct(b['relative_RMSE_improvement']),ci(b['CI95'])])
 section(12,'Error-history retention',table(['Condition','A RMSE','B/C RMSE','A vs A0','B vs B0','B CI95'],hr)+f'''
Magnitude ablation C: {pct(h['arms']['C']['magnitude_ablation']['relative_RMSE_improvement'])}, CI95 {ci(h['arms']['C']['magnitude_ablation']['CI95'])}. Conditional permutation C: {pct(h['arms']['C']['conditional_permutation']['relative_RMSE_improvement'])}, CI95 {ci(h['arms']['C']['conditional_permutation']['CI95'])}.

IID BI vs BI0: {pct(iid['relative_RMSE_improvement'])}, CI90 {ci(iid['CI90'])}; praktis equivalence memerlukan seluruh interval di±3%. Nearest-neighbor permutation target-blind memakai36 observed neurons+previous state+forecast, exact discrete context dan12 donor neighbors dari stream lain. Conditioning ini aproksimatif. Source R5 benefit21.45% adalah referensi, bukan nilai yang dipaksakan ke R6.''')
 cr=[]
 for arm in ['A','B','C','F']:
  z=h['arms'][arm]['pooled'];cr.append([arm,f"{z['mean_actual']:.8f}",f"{z['mean_prediction']:.8f}",f"{z['mean_ratio']:.5f}",f"{z['RMSE']:.8f}",f"{z['constant_Fit_RMSE']:.8f}"])
 section(13,'Contextual calibration',table(['Arm','Mean actual','Mean prediction','Ratio','Native RMSE','Fit-constant RMSE'],cr)+'''
Tidak ada global scaling/bias yang dipasang ke runtime. Mean ratio bukan seluruh definisi calibration; operation groups, calibration bins, residual quantiles, MAE dan per-seed measurements juga disimpan. Calibration gate tetap memerlukan historical retention. Utility tidak dianggap memperbaiki calibration hanya karena query baru dapat dilatih.''')
 rr=[]
 for arm in ARMS:
  z=r['retention'][arm];v=z['pass_counts'];rr.append([arm,f"{v['historical']}/6",f"{v['normal']}/6",f"{v['shifted']}/6",f"{v['strict_S1']}/6",f"{v['AE1']}/6",f"{sum(a['all_retention_pass'] for a in z['per_seed'])}/6"])
 section(14,'Historical knowledge retention',table(['Arm','Historical S2','New normal','New shifted','Strict S1','A–E1','All'],rr)+f'''
Historical S2 limit0.01766457. Frozen independent-development source limits: normal {ret['normal']['limit']:.8f}, shifted {ret['shifted']['limit']:.8f}; masing-masing source RMSE+1e-7 pada128 fresh development operations, sebelum training. Source witnesses128 tidak dihapus; original anchor64, normal64 dan shifted64 dipertahankan sebagai bank terpisah. Per-bank veto counts ada dalam receipts.

Sampling variation source sendiri pada final dilaporkan pada per-seed retention data; threshold tidak dinaikkan setelah melihat hasil. Coverage berbeda tidak dirata-ratakan sehingga satu kelompok menutupi regresi lainnya. Perbaikan bank adalah safeguard, bukan learned retention.''')
 section(15,'Generalization','''Utility seen/unseen/shifted dinilai pada32 independent streams×16 actual operations×dua interventions per model. Holdout tidak digunakan untuk checkpoint selection, fitting atau training lanjut. Transfer di luar numerical-readout pathways, novel magnitudes dan novel continual-learning tasks belum diuji.

Descriptor pasangan memakai mean summaries sehingga dapat kehilangan correspondence antara activity/weight/delta. Sembilan raw coordinates tidak menjamin seluruh konteks/history tersedia. Native activation-basis alias counts di receipts merupakan bukti batas representasi yang diuji, bukan bukti mathematical capacity insufficiency. Feature cap64 tidak dicoba.''')
 section(16,'Causal ablation','''H1 menggunakan paired signed interventions, pathway ablation, held-out pathways, untrained-zero baseline dan F shuffled-label control. Hasil aktual tidak diberikan sebelum utility forecast. History mempunyai magnitude masking dan target-blind conditional permutation.

Tidak ada causal ablation H3/H4 karena fungsi selection belum ada. S2 C dan B identik by design; hasil history/retention tidak boleh diatribusikan kepada utility. Solver, coverage dan representation refinement tetap penjelasan terpisah. Interference-information removal belum menjadi uji yang valid karena H2 belum dipelajari.''')
 controls=[]
 for name,v in u.items():controls.append([name,f"{v['C']['rmse']:.8f}",f"{v['F']['rmse']:.8f}",pct(v['F']['benefit']),str(v['C']['class_counts_negative_neutral_positive'])])
 section(17,'Negative controls',table(['Holdout','C RMSE','F shuffled RMSE','F benefit','C class counts −/0/+'],controls)+'''
F merandomisasi relasi input→utility dengan permutasi label aktual terpisah pada Fit dan Check, menjaga marginal labels. Random labels/shuffled outcomes merupakan satu operationalization, bukan dua eksperimen independen. Konflik akibat random labels tetap dilaporkan dan dapat mengurangi admission; tidak dianggap bukti causal regulation. Activation-only/weight-only/constant/zero controls tidak dipasang ke N.''')
 pr=[]
 for arm in ['A','B','BI','C','F']:
  z=resource[arm];pr.append([arm,z['candidate_trials'],f"{z['learning_nanoseconds_median']/1e9:.3f}",f"{z['inference_p50_nanoseconds']/1e6:.4f}",f"{z['inference_p95_nanoseconds']/1e6:.4f}",f"{z.get('utility_p50_nanoseconds',0)/1e6:.4f}",f"{z['WAL_suffix_bytes_median']/1048576:.3f}"])
 section(18,'Performance and resource costs',table(['Arm','Trials','Median learning s/lesson','S2 p50 ms','S2 p95 ms','Utility p50 ms','Median WAL growth MiB'],pr)+'''
Learning time mencakup native search/admission/journal transaction. Utility latency hanya native utility read, bukan keseluruhan counterfactual acquisition atau ledger sync. Seluruh per-call inference durations, memory snapshots VmRSS/VmHWM, heap counters, actual inventories, checkpoint sizes, full WAL bytes dan recovery costs disimpan. Host dipakai bersama tiga research processes, GOMAXPROCS2 per proses; angka bukan CPU-isolated benchmark atau Android speed.

Exact neural operation counts untuk seluruh instrumentation tidak diukur; candidate counts dan circuit/synapse deltas bukan pengganti FLOP count. Active-neuron runtime work tercatat pada environment/WAL untuk operasi akuisisi. Android arm64 service/chat cross-build lulus; perangkat fisik dan cross-architecture exact replay tidak diuji.

Brain knowledge berada pada canonical neurons/signed synapses/topology. Transient optimization evidence berada pada workspace. Retention witnesses/banks terpisah dari knowledge. Raw experiences dan research logs dipertahankan untuk audit, full WAL untuk recovery; inference utility tidak membaca tabel seluruh pengalaman mentah.''')
 replay=[v['persistence']['replay_nanoseconds'] for z in r['retention'].values() for v in z['per_seed']]
 section(19,'Persistence and recovery',f'''**48/48** final models lolos exact restart, full suffix replay80735→final, canonical/signed reconstruction, original rollback dan unchanged historical WAL prefix. Median full-suffix recovery {np.median(replay)/1e9:.3f} s. Suffix replay mengeksekusi ulang native learning; jangan menyamakannya dengan checkpoint-only restoration.

Prototype V1 dan V2 masing-masing empat C/F development models juga diverifikasi exact restart/full suffix. Kedua protocol tetap didukung. Tests/build/vet, tagged metric/native regressions, targeted race dan Android arm64 cross-build lulus. Unit baru menguji new-query fixed-basis recovery pada kedua protocol, old-function exactness dan checkpoint/full-suffix reconstruction.

Arsip recovery menyimpan source/intermediate/final checkpoints, full historical+new WAL, pre-operation/ intervention ledgers, actual poststates, utility predictions, eligibility, signed deltas, candidates/rejections, witness banks, locks, source hashes, negative development dan complete R5 source archive. `RESTORE.md`, `BUNDLE.json` dan manifest SHA256 menjelaskan recovery. Pengunduhan kembali/hash luar disajikan dalam owner delivery note.''')
 section(20,'Failed experiments','''1. V1 development: unconstrained metric fitting mencapai coefficient di luar signed bound; activation fitting tidak membaik pada independent Check. Semua receipts/checkpoints/WAL tetap ada.
2. V2 development: bounded solver menghilangkan ketergantungan pada satu unconstrained solution, tetapi utility belum mengalahkan baseline independent seen/unseen. Tidak ada threshold yang dilonggarkan.
3. Beberapa lesson C/F diterima pada Check; final evaluation diperlukan untuk menguji apakah gain bertahan. Conflicting/shuffled labels dan no-check-improvement tetap dihitung.
4. H2/H3/H4 serta G dihentikan oleh preregistered development prerequisite, bukan oleh source corruption. Tidak ada hasil yang direka untuk tahap yang tidak dijalankan.
5. Descriptor summaries, fixed existing basis, finite optimizer budget, approximate permutation dan limited new-distribution contract tetap batas inferensi. Tidak satu pun membuktikan cap32 mustahil secara matematis.''')
 chunks[-1]+='\n6. Analyzer semula terhenti pada `trials:null`. Companion compatibility membaca nilai itu sebagai nol kandidat dalam memori, tetap menghitung rejected opportunity, tanpa mengubah source analyzer yang disegel, raw data, kriteria atau hasil native. Traceback dan penjelasan disimpan pada `ANALYSIS_COMPATIBILITY.md`.\n'
 section(21,'Scientific decision per gate',table(['Gate','Status','Makna'],[[k,v['status'],v['basis'] if isinstance(v['basis'],str) else 'Lihat metrik lengkap pada SCIENTIFIC_RESULTS.json'] for k,v in r['gates'].items()])+'''
Repository standing evaluation: tujuan utility diidentifikasi; implementasi native berjalan; hasil dibandingkan dengan ambang/kontrol yang telah dibekukan; keputusan perlu perbaikan. Keberhasilan build/recovery tidak diubah menjadi keberhasilan learned utility. Tidak diperlukan klaim AGI untuk menerima mekanisme yang benar-benar lulus, tetapi gate wajib yang gagal tetap menahan overall PASS.''')
 section(22,'Next research recommendation','''Pertahankan32 fitur dan sumber R2. Prioritas berikutnya adalah menguji sufficiency/access descriptor: pertahankan pasangan activity–weight–delta secara lokal, sertakan konteks/history yang benar-benar tersedia, lalu ukur alias raw/activation/metric secara terpisah pada development baru. Bedakan informasi hilang, akses hilang, basis yang tidak memadai, finite optimizer failure dan admission yang berlebihan; jangan langsung memperbesar kapasitas.

Setelah representasi memadai, uji generic bounded/regularized fitting dengan Fit/Check/dev baru dan pertahankan semua hasil negatif. Utility baru boleh memengaruhi plasticity sesudah H1 dan H2 lolos development independen. Jangan mengubah source/witness limits berdasarkan final R6, jangan memilih seed terbaik, dan jangan mempromosikan salah satu arm ini.

Laporan dan arsip ini mempertahankan kemajuan implementasi sekaligus batasnya: pengalaman causal numerical utility dapat disalurkan ke N dan direcover persis; kemampuan reliable learned retention/adaptive plasticity belum terbukti.''')
 text='\n'.join(chunks)
 spans=re.split(r'(`[^`]*`)',text)
 for i in range(0,len(spans),2):
  spans[i]=re.sub(r'(?<=[a-z])(?=\d)',' ',spans[i])
  spans[i]=re.sub(r'(?<=\d)(?=[a-df-z])',' ',spans[i])
  spans[i]=re.sub(r'([,;:])(?=\d)',r'\1 ',spans[i])
  spans[i]=re.sub(r'(?<=[a-z])(?=[≥≤])',' ',spans[i])
 (out/'S2-R6_NATIVE_NEURAL_UTILITY_RETENTION_LEARNING.md').write_text(''.join(spans))
 shutil.copyfile(w/'SCIENTIFIC_RESULTS.json',out/'SCIENTIFIC_RESULTS.json')
 # Publication figures render exact saved values, without new analysis.
 fig,axes=plt.subplots(1,2,figsize=(12,4.5));labels=['seen','unseen','shifted'];xx=np.arange(3)
 axes[0].bar(xx-.2,[u[v]['C']['rmse'] for v in labels],.4,label='Native utility C');axes[0].bar(xx+.2,[min(u[v]['C']['baseline_rmse'].values()) for v in labels],.4,label='Best Fit-only baseline');axes[0].set_xticks(xx,labels);axes[0].set_ylabel('Utility RMSE');axes[0].legend(fontsize=8);axes[0].set_title('Independent utility prediction')
 aa=['A','B','C','F'];axes[1].bar(aa,[h['arms'][v]['pooled']['RMSE'] for v in aa]);axes[1].set_ylabel('Persistent S2 RMSE');axes[1].set_title('History remains separate from utility')
 fig.suptitle(r['verdict'],fontsize=12);fig.tight_layout();fig.savefig(out/'S2R6_RESULTS.png',dpi=180);plt.close(fig)
 fig,ax=plt.subplots(figsize=(10,3));cols=['historical','normal','shifted','strict_S1','AE1'];mat=np.array([[r['retention'][arm]['pass_counts'][v] for v in cols] for arm in aa]);im=ax.imshow(mat,vmin=0,vmax=6,cmap='YlGn',aspect='auto');ax.set_xticks(range(len(cols)),['Historical S2','New normal','New shifted','Strict S1','A–E1']);ax.set_yticks(range(4),aa)
 for i in range(4):
  for j in range(len(cols)):ax.text(j,i,f'{mat[i,j]}/6',ha='center',va='center')
 ax.set_title('Frozen retention contracts — every seed retained');fig.tight_layout();fig.savefig(out/'S2R6_RETENTION.png',dpi=180);plt.close(fig)
 print(out/'S2-R6_NATIVE_NEURAL_UTILITY_RETENTION_LEARNING.md')
if __name__=='__main__':main()
