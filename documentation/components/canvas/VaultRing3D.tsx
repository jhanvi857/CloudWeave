'use client'

import React, { useRef, useState, useMemo, useEffect } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, Html, Float } from '@react-three/drei'
import * as THREE from 'three'
import { Play, Pause, Key } from 'lucide-react'

// 6 Cluster Nodes (Vault Tumblers)
export interface TumblerNode {
  id: number
  name: string
  angleDeg: number
  role: 'Primary' | 'Replica' | 'Standby'
  vnodes: number
  capacity: string
  chunksStored: number
  tokenRange: string
  status: 'healthy' | 'locking' | 'replicated'
}

const initialTumblers: TumblerNode[] = [
  { id: 1, name: 'Tumbler-01', angleDeg: 0, role: 'Replica', vnodes: 150, capacity: '2.4 GB / 10 GB', chunksStored: 42, tokenRange: '0x0000..0x2AAA', status: 'healthy' },
  { id: 2, name: 'Tumbler-02', angleDeg: 60, role: 'Primary', vnodes: 150, capacity: '2.8 GB / 10 GB', chunksStored: 48, tokenRange: '0x2AAB..0x5555', status: 'healthy' },
  { id: 3, name: 'Tumbler-03', angleDeg: 120, role: 'Replica', vnodes: 150, capacity: '2.2 GB / 10 GB', chunksStored: 39, tokenRange: '0x5556..0x7FFF', status: 'healthy' },
  { id: 4, name: 'Tumbler-04', angleDeg: 180, role: 'Standby', vnodes: 150, capacity: '1.9 GB / 10 GB', chunksStored: 34, tokenRange: '0x8000..0xAAAA', status: 'healthy' },
  { id: 5, name: 'Tumbler-05', angleDeg: 240, role: 'Standby', vnodes: 150, capacity: '2.1 GB / 10 GB', chunksStored: 37, tokenRange: '0xAAAB..0xD555', status: 'healthy' },
  { id: 6, name: 'Tumbler-06', angleDeg: 300, role: 'Replica', vnodes: 150, capacity: '2.5 GB / 10 GB', chunksStored: 44, tokenRange: '0xD556..0xFFFF', status: 'healthy' },
]

const RING_RADIUS = 3.6

// Radial Tick Marks on the Safe Dial (Garnet Major, Steel Semi, Hairline Minor)
function DialTicks({ count = 90, radius = RING_RADIUS }: { count?: number; radius?: number }) {
  const points = useMemo(() => {
    const lines: THREE.Vector3[][] = []
    for (let i = 0; i < count; i++) {
      const angle = (i * 2 * Math.PI) / count
      const isMajor = i % 15 === 0
      const isSemi = i % 5 === 0
      const innerR = radius - (isMajor ? 0.35 : isSemi ? 0.2 : 0.1)
      const outerR = radius + (isMajor ? 0.15 : 0.05)

      const x1 = Math.cos(angle) * innerR
      const z1 = Math.sin(angle) * innerR
      const x2 = Math.cos(angle) * outerR
      const z2 = Math.sin(angle) * outerR

      lines.push([new THREE.Vector3(x1, 0, z1), new THREE.Vector3(x2, 0, z2)])
    }
    return lines
  }, [count, radius])

  return (
    <group>
      {points.map((line, idx) => {
        const isMajor = idx % 15 === 0
        const isSemi = idx % 5 === 0
        const color = isMajor ? '#7A1F2B' : isSemi ? '#A6A9AE' : '#26262B'
        const geom = new THREE.BufferGeometry().setFromPoints(line)
        return (
          <primitive key={idx} object={new THREE.Line(geom, new THREE.LineBasicMaterial({ color, transparent: true, opacity: isMajor ? 0.95 : 0.45 }))} />
        )
      })}
    </group>
  )
}

// 3D Vault Tumbler (Storage Node)
function TumblerMesh({
  tumbler,
  isSelected,
  onClick,
}: {
  tumbler: TumblerNode
  isSelected: boolean
  onClick: () => void
}) {
  const rad = (tumbler.angleDeg * Math.PI) / 180
  const x = Math.cos(rad) * RING_RADIUS
  const z = Math.sin(rad) * RING_RADIUS
  const meshRef = useRef<THREE.Group>(null)

  const isPrimary = tumbler.role === 'Primary'
  const isReplica = tumbler.role === 'Replica'

  return (
    <group ref={meshRef} position={[x, 0, z]}>
      {/* Base Steel/Garnet Chamber */}
      <mesh onClick={(e) => { e.stopPropagation(); onClick() }}>
        <cylinderGeometry args={[0.42, 0.46, 0.5, 32]} />
        <meshStandardMaterial
          color={isSelected ? '#EDE8DF' : isPrimary ? '#7A1F2B' : isReplica ? '#A6A9AE' : '#19191C'}
          metalness={0.85}
          roughness={0.25}
          emissive={isSelected ? '#7A1F2B' : isPrimary ? '#421118' : '#000000'}
          emissiveIntensity={isSelected ? 0.9 : isPrimary ? 0.6 : 0.1}
        />
      </mesh>

      {/* Outer Locking Bezel Ring */}
      <mesh position={[0, 0.26, 0]}>
        <torusGeometry args={[0.44, 0.04, 16, 32]} />
        <meshStandardMaterial color="#26262B" metalness={0.9} roughness={0.2} />
      </mesh>

      {/* Top Spindle Notch */}
      <mesh position={[0, 0.28, 0]}>
        <cylinderGeometry args={[0.22, 0.22, 0.12, 16]} />
        <meshStandardMaterial
          color={isPrimary ? '#7A1F2B' : '#A6A9AE'}
          metalness={0.9}
          roughness={0.15}
        />
      </mesh>

      {/* Locking Bolt Pins */}
      {[-0.32, 0.32].map((offset, i) => (
        <mesh key={i} position={[offset, 0, 0]} rotation={[0, 0, Math.PI / 2]}>
          <cylinderGeometry args={[0.06, 0.06, 0.18, 12]} />
          <meshStandardMaterial color="#A6A9AE" metalness={0.9} roughness={0.1} />
        </mesh>
      ))}

      {/* Node Tag HUD */}
      <Html position={[0, 0.75, 0]} center distanceFactor={12}>
        <button
          onClick={onClick}
          className={`flex flex-col items-center px-2 py-1 rounded border font-mono text-[10px] whitespace-nowrap transition-all shadow-md cursor-pointer ${isSelected
              ? 'bg-[#7A1F2B] text-[#EDE8DF] border-[#EDE8DF] font-bold ring-1 ring-[#EDE8DF]'
              : isPrimary
                ? 'bg-[#19191C]/95 text-[#EDE8DF] border-[#7A1F2B]'
                : isReplica
                  ? 'bg-[#19191C]/95 text-[#A6A9AE] border-[#26262B]'
                  : 'bg-[#19191C]/80 text-[#8B8B8F] border-[#26262B]'
            }`}
        >
          <div className="flex items-center gap-1 font-bold">
            <span className={`w-1.5 h-1.5 rounded-full ${isPrimary ? 'bg-[#7A1F2B]' : isReplica ? 'bg-[#A6A9AE]' : 'bg-[#26262B]'}`} />
            <span>N{tumbler.id}</span>
            <span className="text-[8px] opacity-75 font-normal">({tumbler.role})</span>
          </div>
          <div className="text-[8px] text-[#8B8B8F] mt-0.5">
            {tumbler.chunksStored} chunks
          </div>
        </button>
      </Html>
    </group>
  )
}

// Data Light Trail & Forking Combination Particle
function VaultCombinationStream({
  isPlaying,
  speed,
}: {
  isPlaying: boolean
  speed: number
}) {
  const packetRef = useRef<THREE.Mesh>(null)
  const replica1Ref = useRef<THREE.Mesh>(null)
  const replica2Ref = useRef<THREE.Mesh>(null)
  const progress = useRef(0)

  const sourcePos = useMemo(() => new THREE.Vector3(-5.2, 1.8, 1.2), [])
  const centerPos = useMemo(() => new THREE.Vector3(0, 0.2, 0), [])

  const rad2 = (60 * Math.PI) / 180
  const rad3 = (120 * Math.PI) / 180
  const rad6 = (300 * Math.PI) / 180

  const node2Pos = useMemo(() => new THREE.Vector3(Math.cos(rad2) * RING_RADIUS, 0.2, Math.sin(rad2) * RING_RADIUS), [rad2])
  const node3Pos = useMemo(() => new THREE.Vector3(Math.cos(rad3) * RING_RADIUS, 0.2, Math.sin(rad3) * RING_RADIUS), [rad3])
  const node6Pos = useMemo(() => new THREE.Vector3(Math.cos(rad6) * RING_RADIUS, 0.2, Math.sin(rad6) * RING_RADIUS), [rad6])

  useFrame((_, delta) => {
    if (!isPlaying) return
    progress.current = (progress.current + delta * 0.35 * speed) % 1

    const t = progress.current

    if (t < 0.4) {
      const subT = t / 0.4
      const p = new THREE.Vector3().lerpVectors(sourcePos, centerPos, subT)
      if (packetRef.current) {
        packetRef.current.position.copy(p)
        packetRef.current.visible = true
      }
      if (replica1Ref.current) replica1Ref.current.visible = false
      if (replica2Ref.current) replica2Ref.current.visible = false
    } else {
      const subT = (t - 0.4) / 0.6

      const pPrimary = new THREE.Vector3().lerpVectors(centerPos, node2Pos, subT)
      const pReplica1 = new THREE.Vector3().lerpVectors(centerPos, node3Pos, subT)
      const pReplica2 = new THREE.Vector3().lerpVectors(centerPos, node6Pos, subT)

      if (packetRef.current) {
        packetRef.current.position.copy(pPrimary)
        packetRef.current.visible = true
      }
      if (replica1Ref.current) {
        replica1Ref.current.position.copy(pReplica1)
        replica1Ref.current.visible = true
      }
      if (replica2Ref.current) {
        replica2Ref.current.position.copy(pReplica2)
        replica2Ref.current.visible = true
      }
    }
  })

  return (
    <group>
      {/* Primary Light Packet */}
      <mesh ref={packetRef}>
        <sphereGeometry args={[0.12, 16, 16]} />
        <meshStandardMaterial
          color="#EDE8DF"
          emissive="#7A1F2B"
          emissiveIntensity={3.5}
          roughness={0.1}
        />
      </mesh>

      {/* Forked Replica 1 Packet */}
      <mesh ref={replica1Ref} visible={false}>
        <sphereGeometry args={[0.1, 16, 16]} />
        <meshStandardMaterial
          color="#EDE8DF"
          emissive="#7A1F2B"
          emissiveIntensity={2.5}
          roughness={0.1}
        />
      </mesh>

      {/* Forked Replica 2 Packet */}
      <mesh ref={replica2Ref} visible={false}>
        <sphereGeometry args={[0.1, 16, 16]} />
        <meshStandardMaterial
          color="#EDE8DF"
          emissive="#A6A9AE"
          emissiveIntensity={2.2}
          roughness={0.1}
        />
      </mesh>
    </group>
  )
}

// Ingested Crystalline File Payload
function FloatingFileObject() {
  return (
    <Float speed={1.5} rotationIntensity={0.6} floatIntensity={0.5} position={[-5.2, 1.8, 1.2]}>
      <group>
        <mesh rotation={[Math.PI / 4, 0, Math.PI / 4]}>
          <octahedronGeometry args={[0.55, 0]} />
          <meshStandardMaterial
            color="#19191C"
            emissive="#7A1F2B"
            emissiveIntensity={0.5}
            metalness={0.9}
            roughness={0.2}
          />
        </mesh>

        <mesh rotation={[Math.PI / 4, 0, Math.PI / 4]}>
          <octahedronGeometry args={[0.57, 0]} />
          <meshStandardMaterial
            color="#A6A9AE"
            wireframe
            transparent
            opacity={0.6}
          />
        </mesh>

        <Html position={[0, -0.7, 0]} center distanceFactor={10}>
          <div className="px-2 py-1 rounded bg-[#000000]/95 border border-[#26262B] font-mono text-[9px] text-[#EDE8DF] whitespace-nowrap shadow-sm">
            <div className="font-bold text-[#EDE8DF]">S3 Object</div>
            <div className="text-[8px] text-[#8B8B8F]">video.mp4 (2.4 GB)</div>
          </div>
        </Html>
      </group>
    </Float>
  )
}

// Main Vault Dial Mechanism Assembly
function VaultDialScene({
  isPlaying,
  speed,
  selectedTumbler,
  setSelectedTumbler,
}: {
  isPlaying: boolean
  speed: number
  selectedTumbler: number | null
  setSelectedTumbler: (id: number | null) => void
}) {
  const dialAssemblyRef = useRef<THREE.Group>(null)

  useFrame((_, delta) => {
    if (dialAssemblyRef.current && isPlaying) {
      dialAssemblyRef.current.rotation.y += delta * 0.05 * speed
    }
  })

  return (
    <group>
      <ambientLight intensity={0.5} />
      <directionalLight position={[6, 10, 5]} intensity={1.2} color="#EDE8DF" />
      <directionalLight position={[-8, 6, -4]} intensity={0.8} color="#7A1F2B" />
      <pointLight position={[0, 2, 0]} intensity={1.8} color="#7A1F2B" distance={8} />

      <FloatingFileObject />

      <group ref={dialAssemblyRef}>
        {/* Base Cylinder */}
        <mesh position={[0, -0.25, 0]}>
          <cylinderGeometry args={[RING_RADIUS + 0.9, RING_RADIUS + 1.1, 0.45, 64]} />
          <meshStandardMaterial
            color="#19191C"
            metalness={0.85}
            roughness={0.3}
          />
        </mesh>

        {/* Outer Steel Bezel Ring */}
        <mesh position={[0, 0.02, 0]}>
          <torusGeometry args={[RING_RADIUS + 0.6, 0.08, 16, 64]} />
          <meshStandardMaterial
            color="#A6A9AE"
            metalness={0.9}
            roughness={0.15}
          />
        </mesh>

        {/* Inner Engraved Ring */}
        <mesh position={[0, 0.02, 0]}>
          <torusGeometry args={[RING_RADIUS, 0.04, 16, 64]} />
          <meshStandardMaterial
            color="#26262B"
            metalness={0.7}
            roughness={0.4}
          />
        </mesh>

        <DialTicks count={90} radius={RING_RADIUS} />

        {/* Central Spindle Hub */}
        <group position={[0, 0, 0]}>
          <mesh position={[0, 0.15, 0]}>
            <cylinderGeometry args={[0.9, 1.1, 0.35, 32]} />
            <meshStandardMaterial
              color="#19191C"
              metalness={0.9}
              roughness={0.2}
              emissive="#421118"
              emissiveIntensity={0.3}
            />
          </mesh>
          <mesh position={[0, 0.34, 0]}>
            <cylinderGeometry args={[0.5, 0.5, 0.12, 24]} />
            <meshStandardMaterial
              color="#7A1F2B"
              metalness={0.95}
              roughness={0.1}
            />
          </mesh>
          {[0, Math.PI / 2].map((rot, i) => (
            <mesh key={i} position={[0, 0.36, 0]} rotation={[0, rot, 0]}>
              <boxGeometry args={[1.3, 0.06, 0.08]} />
              <meshStandardMaterial color="#A6A9AE" metalness={0.9} roughness={0.1} />
            </mesh>
          ))}

          <Html position={[0, 0.65, 0]} center distanceFactor={10}>
            <div className="px-2 py-0.5 rounded bg-[#000000]/95 border border-[#7A1F2B] text-[9px] font-mono text-[#EDE8DF] font-bold shadow-sm whitespace-nowrap">
              SHA-256 CAS Router
            </div>
          </Html>
        </group>

        {initialTumblers.map((tumbler) => (
          <TumblerMesh
            key={tumbler.id}
            tumbler={tumbler}
            isSelected={selectedTumbler === tumbler.id}
            onClick={() => setSelectedTumbler(selectedTumbler === tumbler.id ? null : tumbler.id)}
          />
        ))}
      </group>

      <VaultCombinationStream isPlaying={isPlaying} speed={speed} />

      <OrbitControls
        enablePan={false}
        enableZoom={true}
        minDistance={6}
        maxDistance={14}
        maxPolarAngle={Math.PI / 2 - 0.05}
        minPolarAngle={Math.PI / 6}
        dampingFactor={0.05}
      />
    </group>
  )
}

export function VaultRing3D() {
  const [isPlaying, setIsPlaying] = useState<boolean>(true)
  const [speed, setSpeed] = useState<number>(1.0)
  const [selectedTumblerId, setSelectedTumblerId] = useState<number | null>(2)
  const [prefersReducedMotion, setPrefersReducedMotion] = useState<boolean>(false)

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    setPrefersReducedMotion(mediaQuery.matches)
    const handler = (e: MediaQueryListEvent) => setPrefersReducedMotion(e.matches)
    mediaQuery.addEventListener('change', handler)
    return () => mediaQuery.removeEventListener('change', handler)
  }, [])

  const selectedNode = initialTumblers.find((t) => t.id === selectedTumblerId) || initialTumblers[1]

  return (
    <div className="relative w-full h-[520px] sm:h-[600px] lg:h-[680px] bg-[#000000] border border-[#26262B] rounded-xl overflow-hidden select-none font-sans">
      {prefersReducedMotion ? (
        <div className="w-full h-full flex flex-col items-center justify-center p-8 text-center bg-[#19191C]">
          <div className="w-48 h-48 rounded-full border-2 border-dashed border-[#7A1F2B] flex items-center justify-center mb-4">
            <div className="text-center font-mono">
              <div className="text-lg font-bold text-[#EDE8DF]">Consistent Hash Ring</div>
              <div className="text-xs text-[#8B8B8F]">6 Cluster Nodes</div>
            </div>
          </div>
          <p className="text-xs font-mono text-[#8B8B8F] max-w-sm">
            Static view active. 150 virtual nodes balance SHA-256 CAS chunks with N=3 quorum.
          </p>
        </div>
      ) : (
        <Canvas
          camera={{ position: [0, 6.5, 8.5], fov: 42 }}
          gl={{ antialias: true, alpha: false }}
          className="w-full h-full cursor-grab active:cursor-grabbing"
        >
          <color attach="background" args={['#000000']} />
          <VaultDialScene
            isPlaying={isPlaying}
            speed={speed}
            selectedTumbler={selectedTumblerId}
            setSelectedTumbler={setSelectedTumblerId}
          />
        </Canvas>
      )}

      {/* Top Left HUD */}
      <div className="absolute top-4 left-4 z-10 flex flex-col gap-2 font-mono text-xs">
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-md bg-[#19191C]/90 border border-[#26262B] backdrop-blur-md text-[#EDE8DF]">
          <div className="w-2 h-2 rounded-full bg-[#7A1F2B] animate-pulse" />
          <span className="font-bold text-[#EDE8DF]">CONSISTENT HASH RING</span>
          <span className="text-[#A6A9AE] text-[11px]">· 150 Vnodes/Node</span>
        </div>
      </div>

      {/* Top Right Controls */}
      <div className="absolute top-4 right-4 z-10 flex items-center gap-2 font-mono text-xs">
        <button
          onClick={() => setIsPlaying(!isPlaying)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#19191C]/90 border border-[#26262B] hover:border-[#7A1F2B] text-[#EDE8DF] transition-colors cursor-pointer"
          title={isPlaying ? 'Pause' : 'Play'}
        >
          {isPlaying ? <Pause className="w-3.5 h-3.5 text-[#A6A9AE]" /> : <Play className="w-3.5 h-3.5 text-[#A6A9AE]" />}
          <span className="text-[11px]">{isPlaying ? 'Pause' : 'Play'}</span>
        </button>

        <div className="flex items-center rounded-md bg-[#19191C]/90 border border-[#26262B] p-0.5">
          {[0.5, 1.0, 2.0].map((s) => (
            <button
              key={s}
              onClick={() => setSpeed(s)}
              className={`px-2 py-1 text-[10px] rounded transition-all cursor-pointer ${speed === s ? 'bg-[#7A1F2B] text-[#EDE8DF] font-bold' : 'text-[#8B8B8F] hover:text-[#EDE8DF]'
                }`}
            >
              {s}x
            </button>
          ))}
        </div>
      </div>

      {/* Bottom Left Inspector Panel */}
      <div className="absolute bottom-4 left-4 z-10 max-w-xs font-mono text-xs">
        <div className="p-3 rounded-lg bg-[#19191C]/95 border border-[#26262B] backdrop-blur-md space-y-2 shadow-lg">
          <div className="flex items-center justify-between border-b border-[#26262B] pb-1.5">
            <div className="flex items-center gap-1.5">
              <Key className="w-3.5 h-3.5 text-[#A6A9AE]" />
              <span className="font-bold text-[#EDE8DF]">{selectedNode.name}</span>
            </div>
            <span className={`text-[10px] px-1.5 py-0.5 rounded font-semibold ${selectedNode.role === 'Primary' ? 'bg-[#7A1F2B] text-[#EDE8DF]' : 'bg-[#26262B] text-[#A6A9AE]'
              }`}>
              {selectedNode.role}
            </span>
          </div>

          <div className="grid grid-cols-2 gap-2 text-[10px] text-[#8B8B8F]">
            <div>
              <span className="block text-[#8B8B8F]">Token Span:</span>
              <span className="text-[#EDE8DF] font-semibold">{selectedNode.tokenRange}</span>
            </div>
            <div>
              <span className="block text-[#8B8B8F]">Disk Storage:</span>
              <span className="text-[#EDE8DF] font-semibold">{selectedNode.capacity}</span>
            </div>
            <div>
              <span className="block text-[#8B8B8F]">Virtual Nodes:</span>
              <span className="text-[#EDE8DF] font-semibold">{selectedNode.vnodes} positions</span>
            </div>
            <div>
              <span className="block text-[#8B8B8F]">Stored Chunks:</span>
              <span className="text-[#EDE8DF] font-semibold">{selectedNode.chunksStored} CAS blocks</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
