'use client'

import React, { useState, useRef } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, Html } from '@react-three/drei'
import * as THREE from 'three'
import { Cpu, RefreshCw, Zap } from 'lucide-react'

export interface ShardBlock {
  id: string
  name: string
  type: 'data' | 'parity'
  node: string
  alive: boolean
  xPos: number
}

// 3D Shard Cube Block
function ShardCube({
  shard,
  isReconstructing,
  onToggle,
}: {
  shard: ShardBlock
  isReconstructing: boolean
  onToggle: () => void
}) {
  const meshRef = useRef<THREE.Group>(null)

  const isData = shard.type === 'data'

  useFrame((_, delta) => {
    if (meshRef.current) {
      if (!shard.alive) {
        meshRef.current.position.y = THREE.MathUtils.lerp(meshRef.current.position.y, -1.2, delta * 4)
        meshRef.current.rotation.x = THREE.MathUtils.lerp(meshRef.current.rotation.x, 0.3, delta * 4)
      } else {
        meshRef.current.position.y = THREE.MathUtils.lerp(meshRef.current.position.y, 0, delta * 6)
        meshRef.current.rotation.x = THREE.MathUtils.lerp(meshRef.current.rotation.x, 0, delta * 6)
      }
    }
  })

  const baseColor = !shard.alive
    ? '#34343A'
    : isData
      ? '#7A1F2B'
      : '#A6A9AE'

  return (
    <group ref={meshRef} position={[shard.xPos, 0, 0]}>
      <mesh onClick={(e) => { e.stopPropagation(); onToggle() }}>
        <boxGeometry args={[0.75, 1.1, 0.5]} />
        <meshStandardMaterial
          color={baseColor}
          metalness={0.8}
          roughness={0.25}
          emissive={!shard.alive ? '#000000' : isData ? '#421118' : '#26262B'}
          emissiveIntensity={!shard.alive ? 0.2 : 0.4}
          transparent={!shard.alive}
          opacity={!shard.alive ? 0.35 : 1.0}
        />
      </mesh>

      <mesh>
        <boxGeometry args={[0.77, 1.12, 0.52]} />
        <meshStandardMaterial
          color={!shard.alive ? '#7A1F2B' : isData ? '#EDE8DF' : '#A6A9AE'}
          wireframe
          transparent
          opacity={!shard.alive ? 0.3 : 0.5}
        />
      </mesh>

      <Html position={[0, 0.9, 0]} center distanceFactor={10}>
        <button
          onClick={onToggle}
          className={`px-2 py-1 rounded font-mono text-[9px] whitespace-nowrap border transition-all cursor-pointer shadow-md ${!shard.alive
              ? 'bg-[#19191C] text-[#8B8B8F] border-[#7A1F2B]'
              : isData
                ? 'bg-[#19191C] text-[#EDE8DF] border-[#7A1F2B]'
                : 'bg-[#19191C] text-[#A6A9AE] border-[#26262B]'
            }`}
        >
          <div className="font-bold flex items-center gap-1">
            <span>{shard.name}</span>
            <span>{shard.alive ? '✓' : '✗'}</span>
          </div>
          <div className="text-[8px] text-[#8B8B8F]">{shard.node}</div>
        </button>
      </Html>
    </group>
  )
}

function ReconstructionBeams({
  shards,
  isReconstructing,
}: {
  shards: ShardBlock[]
  isReconstructing: boolean
}) {
  if (!isReconstructing) return null

  const surviving = shards.filter((s) => s.alive)
  const failed = shards.filter((s) => !s.alive)

  return (
    <group>
      {failed.map((f) =>
        surviving.slice(0, 4).map((s, idx) => {
          const points = [
            new THREE.Vector3(s.xPos, 0, 0),
            new THREE.Vector3((s.xPos + f.xPos) / 2, 0.6, 0.3),
            new THREE.Vector3(f.xPos, 0, 0),
          ]
          const curve = new THREE.CatmullRomCurve3(points)
          const curvePoints = curve.getPoints(20)
          const geom = new THREE.BufferGeometry().setFromPoints(curvePoints)

          return (
            <primitive
              key={`${f.id}-${s.id}-${idx}`}
              object={
                new THREE.Line(
                  geom,
                  new THREE.LineBasicMaterial({
                    color: '#7A1F2B',
                    linewidth: 2,
                    transparent: true,
                    opacity: 0.85,
                  })
                )
              }
            />
          )
        })
      )}
    </group>
  )
}

export function ErasureCoding3D() {
  const [shards, setShards] = useState<ShardBlock[]>([
    { id: 'd1', name: 'D1 (Data)', type: 'data', node: 'Vault-01', alive: true, xPos: -2.25 },
    { id: 'd2', name: 'D2 (Data)', type: 'data', node: 'Vault-02', alive: true, xPos: -1.35 },
    { id: 'd3', name: 'D3 (Data)', type: 'data', node: 'Vault-03', alive: true, xPos: -0.45 },
    { id: 'd4', name: 'D4 (Data)', type: 'data', node: 'Vault-04', alive: true, xPos: 0.45 },
    { id: 'p1', name: 'P1 (Parity)', type: 'parity', node: 'Vault-05', alive: true, xPos: 1.35 },
    { id: 'p2', name: 'P2 (Parity)', type: 'parity', node: 'Vault-06', alive: true, xPos: 2.25 },
  ])

  const [isReconstructing, setIsReconstructing] = useState<boolean>(false)

  const toggleShard = (id: string) => {
    setShards((prev) =>
      prev.map((s) => (s.id === id ? { ...s, alive: !s.alive } : s))
    )
  }

  const aliveCount = shards.filter((s) => s.alive).length
  const failedCount = shards.length - aliveCount
  const canReconstruct = aliveCount >= 4

  const handleReconstruct = () => {
    if (!canReconstruct) return
    setIsReconstructing(true)
    setTimeout(() => {
      setShards((prev) => prev.map((s) => ({ ...s, alive: true })))
      setIsReconstructing(false)
    }, 1200)
  }

  const handleReset = () => {
    setShards((prev) => prev.map((s) => ({ ...s, alive: true })))
    setIsReconstructing(false)
  }

  return (
    <div className="w-full flex flex-col bg-[#19191C] border border-[#26262B] rounded-xl overflow-hidden font-sans select-none">
      {/* Top Header */}
      <div className="p-4 border-b border-[#26262B] flex flex-wrap items-center justify-between gap-3 font-mono text-xs bg-[#000000]">
        <div className="flex items-center gap-2">
          <Cpu className="w-4 h-4 text-[#A6A9AE]" />
          <span className="font-bold text-[#EDE8DF]">REED-SOLOMON ERASURE CODING (K=4, M=2)</span>
        </div>

        <div className="flex items-center gap-2">
          {failedCount > 0 && (
            <button
              onClick={handleReconstruct}
              disabled={!canReconstruct || isReconstructing}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-[#7A1F2B] text-[#EDE8DF] font-semibold hover:bg-[#912735] disabled:opacity-40 disabled:cursor-not-allowed transition-colors text-xs cursor-pointer shadow-xs"
            >
              <Zap className="w-3.5 h-3.5 fill-current" />
              <span>{isReconstructing ? 'Solving GF(2^8)...' : 'Reconstruct Shards'}</span>
            </button>
          )}

          <button
            onClick={handleReset}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded border border-[#26262B] text-[#8B8B8F] hover:text-[#EDE8DF] hover:border-[#34343A] transition-colors text-xs cursor-pointer"
          >
            <RefreshCw className="w-3 h-3" />
            <span>Reset</span>
          </button>
        </div>
      </div>

      {/* 3D Canvas Area */}
      <div className="w-full h-[360px] sm:h-[400px] relative bg-[#000000]">
        <Canvas camera={{ position: [0, 1.2, 5.0], fov: 45 }}>
          <ambientLight intensity={0.5} />
          <directionalLight position={[4, 6, 4]} intensity={1.2} color="#EDE8DF" />
          <pointLight position={[0, 1, 0]} intensity={1.5} color="#7A1F2B" distance={8} />

          <group position={[0, 0, 0]}>
            {shards.map((shard) => (
              <ShardCube
                key={shard.id}
                shard={shard}
                isReconstructing={isReconstructing}
                onToggle={() => toggleShard(shard.id)}
              />
            ))}
            <ReconstructionBeams shards={shards} isReconstructing={isReconstructing} />
          </group>

          <OrbitControls enableZoom={false} enablePan={false} maxPolarAngle={Math.PI / 2} minPolarAngle={Math.PI / 3} />
        </Canvas>

        <div className="absolute bottom-3 left-3 font-mono text-[10px] text-[#8B8B8F] bg-[#19191C]/80 px-2.5 py-1 rounded border border-[#26262B]">
          Click any 3D shard block to toggle failure
        </div>
      </div>

      {/* Bottom Status */}
      <div className="p-4 border-t border-[#26262B] bg-[#19191C] space-y-4">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-3 rounded-lg border border-[#26262B] bg-[#000000] font-mono text-xs">
          <div className="flex items-center gap-2">
            <div
              className={`w-2.5 h-2.5 rounded-full ${aliveCount === 6
                  ? 'bg-[#EDE8DF]'
                  : canReconstruct
                    ? 'bg-[#A6A9AE]'
                    : 'bg-[#7A1F2B]'
                }`}
            />
            <span className="font-bold text-[#EDE8DF]">
              {aliveCount === 6
                ? 'System Fully Healthy (6/6 Shards Online)'
                : canReconstruct
                  ? `Degraded · ${aliveCount}/6 Online (Tolerates ${aliveCount - 4} more failure)`
                  : `CRITICAL DATA LOSS · ${aliveCount}/6 Online (< K=4 Threshold)`}
            </span>
          </div>

          <div className="text-[11px] text-[#8B8B8F]">
            Overhead: <span className="text-[#EDE8DF] font-bold">1.5x</span> (vs 3.0x raw 3-way replication)
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 font-mono text-xs text-[#8B8B8F]">
          <div className="p-2.5 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Matrix Type:</span>
            <span className="text-[#EDE8DF] font-semibold">Cauchy / Vandermonde GF(2^8)</span>
          </div>
          <div className="p-2.5 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Fault Tolerance:</span>
            <span className="text-[#EDE8DF] font-semibold">M=2 Simultaneous Node Deaths</span>
          </div>
          <div className="p-2.5 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Reconstruction Cost:</span>
            <span className="text-[#EDE8DF] font-semibold">Any K=4 out of N=6 Shards</span>
          </div>
        </div>
      </div>
    </div>
  )
}
