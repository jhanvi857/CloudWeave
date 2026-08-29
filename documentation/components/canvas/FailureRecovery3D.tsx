'use client'

import React, { useState, useRef } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, Html } from '@react-three/drei'
import * as THREE from 'three'
import { Activity, AlertOctagon, RefreshCw } from 'lucide-react'

interface ClusterNode3D {
  id: number
  name: string
  pos: [number, number, number]
  status: 'healthy' | 'timeout' | 'failed' | 'repairing'
  chunks: string[]
}

const initialClusterNodes: ClusterNode3D[] = [
  { id: 1, name: 'Vault-01', pos: [-2.2, 0.8, 0], status: 'healthy', chunks: ['Chunk A', 'Chunk B', 'Chunk C'] },
  { id: 2, name: 'Vault-02', pos: [-1.2, -1.2, 0.8], status: 'healthy', chunks: ['Chunk A', 'Chunk B'] },
  { id: 3, name: 'Vault-03', pos: [0, 1.6, -0.6], status: 'healthy', chunks: ['Chunk C', 'Chunk D'] }, // Target victim
  { id: 4, name: 'Vault-04', pos: [2.2, 0.8, 0], status: 'healthy', chunks: ['Chunk B', 'Chunk D'] }, // Repair target
  { id: 5, name: 'Vault-05', pos: [1.2, -1.2, 0.8], status: 'healthy', chunks: ['Chunk A', 'Chunk D'] },
]

// 3D Node Mesh
function ClusterNodeMesh({
  node,
  onKill,
}: {
  node: ClusterNode3D
  onKill: () => void
}) {
  const meshRef = useRef<THREE.Group>(null)

  const isFailed = node.status === 'failed' || node.status === 'timeout'
  const isRepairing = node.status === 'repairing'

  const baseColor = isFailed ? '#7A1F2B' : isRepairing ? '#EDE8DF' : '#A6A9AE'

  useFrame((_, delta) => {
    if (meshRef.current) {
      if (isRepairing) {
        meshRef.current.rotation.y += delta * 2
      } else {
        meshRef.current.rotation.y += delta * 0.2
      }
    }
  })

  return (
    <group ref={meshRef} position={node.pos}>
      <mesh onClick={(e) => { e.stopPropagation(); onKill() }}>
        <cylinderGeometry args={[0.5, 0.55, 0.6, 24]} />
        <meshStandardMaterial
          color={baseColor}
          metalness={0.85}
          roughness={0.2}
          emissive={isFailed ? '#7A1F2B' : isRepairing ? '#26262B' : '#19191C'}
          emissiveIntensity={isFailed ? 0.9 : isRepairing ? 0.6 : 0.2}
        />
      </mesh>

      <mesh position={[0, 0.32, 0]}>
        <torusGeometry args={[0.52, 0.04, 16, 32]} />
        <meshStandardMaterial color={isFailed ? '#7A1F2B' : '#26262B'} metalness={0.9} />
      </mesh>

      <Html position={[0, 0.7, 0]} center distanceFactor={10}>
        <button
          onClick={onKill}
          className={`px-2 py-1 rounded font-mono text-[9px] whitespace-nowrap border transition-all cursor-pointer shadow-md ${isFailed
              ? 'bg-[#7A1F2B] text-[#EDE8DF] border-[#7A1F2B]'
              : isRepairing
                ? 'bg-[#19191C] text-[#EDE8DF] border-[#EDE8DF] ring-1 ring-[#7A1F2B]'
                : 'bg-[#19191C] text-[#EDE8DF] border-[#26262B] hover:border-[#A6A9AE]'
            }`}
        >
          <div className="font-bold flex items-center gap-1">
            <span className={`w-1.5 h-1.5 rounded-full ${isFailed ? 'bg-[#EDE8DF]' : 'bg-[#7A1F2B]'}`} />
            <span>{node.name}</span>
          </div>
          <div className="text-[8px] text-[#8B8B8F]">
            {isFailed ? 'DEAD (kill -9)' : `${node.chunks.length} chunks`}
          </div>
        </button>
      </Html>
    </group>
  )
}

function RepairStreamBeam({
  fromPos,
  toPos,
  active,
}: {
  fromPos: [number, number, number]
  toPos: [number, number, number]
  active: boolean
}) {
  const particleRef = useRef<THREE.Mesh>(null)
  const progress = useRef(0)

  useFrame((_, delta) => {
    if (!active) return
    progress.current = (progress.current + delta * 1.2) % 1
    const p1 = new THREE.Vector3(...fromPos)
    const p2 = new THREE.Vector3(...toPos)
    const current = new THREE.Vector3().lerpVectors(p1, p2, progress.current)
    if (particleRef.current) {
      particleRef.current.position.copy(current)
    }
  })

  if (!active) return null

  const points = [new THREE.Vector3(...fromPos), new THREE.Vector3(...toPos)]
  const geom = new THREE.BufferGeometry().setFromPoints(points)

  return (
    <group>
      <primitive
        object={
          new THREE.Line(
            geom,
            new THREE.LineBasicMaterial({
              color: '#7A1F2B',
              linewidth: 2,
              transparent: true,
              opacity: 0.8,
            })
          )
        }
      />
      <mesh ref={particleRef}>
        <sphereGeometry args={[0.12, 16, 16]} />
        <meshStandardMaterial color="#EDE8DF" emissive="#7A1F2B" emissiveIntensity={3} />
      </mesh>
    </group>
  )
}

export function FailureRecovery3D() {
  const [phase, setPhase] = useState<'idle' | 'lost' | 'failed' | 'repairing' | 'recovered'>('idle')
  const [statusMessage, setStatusMessage] = useState<string>('All 5 cluster nodes reporting 300ms heartbeat OK')
  const [nodes, setNodes] = useState<ClusterNode3D[]>(initialClusterNodes)

  const handleKillNode = (targetId: number = 3) => {
    setPhase('lost')
    setStatusMessage('Node 3 heartbeat lost · 300ms gossip failure detector countdown...')
    setNodes((prev) =>
      prev.map((n) => (n.id === targetId ? { ...n, status: 'timeout' } : n))
    )

    setTimeout(() => {
      setPhase('failed')
      setStatusMessage('Node 3 marked DEAD · Under-replicated chunks identified: [Chunk C (N=2 < 3)]')
      setNodes((prev) =>
        prev.map((n) => (n.id === targetId ? { ...n, status: 'failed', chunks: [] } : n))
      )
    }, 1200)

    setTimeout(() => {
      setPhase('repairing')
      setStatusMessage('Self-healing worker pool copying Chunk C from Vault-01 → Vault-04...')
      setNodes((prev) =>
        prev.map((n) => (n.id === 4 ? { ...n, status: 'repairing' } : n))
      )
    }, 2600)

    setTimeout(() => {
      setPhase('recovered')
      setStatusMessage('Cluster Self-Healed: Chunk C restored on Vault-04 · Replication back to N=3')
      setNodes((prev) =>
        prev.map((n) =>
          n.id === targetId
            ? { ...n, status: 'failed', chunks: [] }
            : n.id === 4
              ? { ...n, status: 'healthy', chunks: ['Chunk B', 'Chunk D', 'Chunk C (Healed)'] }
              : { ...n, status: 'healthy' }
        )
      )
    }, 4500)
  }

  const handleReset = () => {
    setPhase('idle')
    setNodes(initialClusterNodes)
    setStatusMessage('All 5 cluster nodes reporting 300ms heartbeat OK')
  }

  return (
    <div className="w-full flex flex-col bg-[#19191C] border border-[#26262B] rounded-xl overflow-hidden font-sans select-none">
      <div className="p-4 border-b border-[#26262B] flex flex-wrap items-center justify-between gap-3 font-mono text-xs bg-[#000000]">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-[#A6A9AE]" />
          <span className="font-bold text-[#EDE8DF]">CLUSTER FAILURE & SELF-HEALING SIMULATOR</span>
        </div>

        <div className="flex items-center gap-2">
          {phase === 'idle' ? (
            <button
              onClick={() => handleKillNode(3)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-[#7A1F2B] text-[#EDE8DF] font-semibold hover:bg-[#912735] transition-colors text-xs cursor-pointer shadow-xs"
            >
              <AlertOctagon className="w-3.5 h-3.5" />
              <span>Simulate Node Kill (`kill -9`)</span>
            </button>
          ) : (
            <button
              onClick={handleReset}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded border border-[#26262B] text-[#8B8B8F] hover:text-[#EDE8DF] hover:border-[#34343A] transition-colors text-xs cursor-pointer"
            >
              <RefreshCw className="w-3 h-3" />
              <span>Reset Cluster</span>
            </button>
          )}
        </div>
      </div>

      <div className="w-full h-[360px] sm:h-[400px] relative bg-[#000000]">
        <Canvas camera={{ position: [0, 0, 5.2], fov: 45 }}>
          <ambientLight intensity={0.6} />
          <directionalLight position={[5, 6, 5]} intensity={1.2} color="#EDE8DF" />
          <pointLight position={[0, 0, 0]} intensity={1.5} color="#7A1F2B" distance={8} />

          <group>
            {nodes.map((node) => (
              <ClusterNodeMesh
                key={node.id}
                node={node}
                onKill={() => phase === 'idle' && handleKillNode(node.id)}
              />
            ))}

            <RepairStreamBeam
              fromPos={nodes[0].pos}
              toPos={nodes[3].pos}
              active={phase === 'repairing'}
            />
          </group>

          <OrbitControls enableZoom={false} enablePan={false} />
        </Canvas>

        <div className="absolute bottom-3 left-3 font-mono text-[10px] text-[#8B8B8F] bg-[#19191C]/80 px-2.5 py-1 rounded border border-[#26262B]">
          Click any 3D node to simulate crash
        </div>
      </div>

      <div className="p-4 border-t border-[#26262B] bg-[#19191C] space-y-3">
        <div className="flex items-center gap-2 p-2.5 rounded border border-[#26262B] bg-[#000000] font-mono text-xs text-[#EDE8DF]">
          <span className={`w-2 h-2 rounded-full ${phase === 'idle' ? 'bg-[#EDE8DF]' : phase === 'recovered' ? 'bg-[#EDE8DF]' : 'bg-[#7A1F2B]'
            }`} />
          <span className="font-semibold">{statusMessage}</span>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 font-mono text-xs">
          <div className="p-2 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Quorum Health:</span>
            <span className="text-[#EDE8DF] font-bold">100% Available</span>
          </div>
          <div className="p-2 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Gossip Timeout:</span>
            <span className="text-[#EDE8DF] font-semibold">300 ms</span>
          </div>
          <div className="p-2 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Repair Throughput:</span>
            <span className="text-[#EDE8DF] font-semibold">1.42 GB/s</span>
          </div>
          <div className="p-2 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Recovery Duration:</span>
            <span className="text-[#EDE8DF] font-bold">1.18 s</span>
          </div>
        </div>
      </div>
    </div>
  )
}
